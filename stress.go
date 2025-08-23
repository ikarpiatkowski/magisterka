package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// === Parametry konfiguracyjne ===
const (
    targetDuration = 10 * time.Minute    // Czas trwania testu
    numGoroutines  = 16                 // Liczba równoległych Goroutines na bazę
    batchSize      = 30                 // Rozmiar operacji pakietowych
    dbHost         = "localhost"
    rampUpDuration = 300 * time.Second   // Czas na stopniowe zwiększanie obciążenia
)

// Dostępne bazy: 'postgres', 'mongo', 'es'
// Usuń z listy te, których nie chcesz testować.
var databasesToTest = []string{"mongo"}

// === Struktury i funkcje pomocnicze ===

// TestResult przechowuje wyniki dla pojedynczego testu bazy danych.
type TestResult struct {
    dbName     string
    duration   time.Duration
    operations int64
    errors     int64
}

// === PostgreSQL ===
func runPostgresTest(ctx context.Context) TestResult {
    log.Println("PostgreSQL: Rozpoczynam test CRUD...")

    connString := fmt.Sprintf("postgres://admin:password123@%s:5432/testdb?pool_max_conns=%d", dbHost, numGoroutines)
    pool, err := pgxpool.New(ctx, connString)
    if err != nil {
        log.Printf("PostgreSQL: Błąd połączenia z pulą: %v", err)
        return TestResult{dbName: "Postgres", errors: -1}
    }
    defer pool.Close()

    // Oczyszczenie tabeli przed testem
    if _, err := pool.Exec(ctx, "TRUNCATE my_data"); err != nil {
        log.Printf("PostgreSQL: Błąd TRUNCATE: %v", err)
    }

    var ops int64
    var errors int64
    var wg sync.WaitGroup

    startTime := time.Now()
    testCtx, cancel := context.WithTimeout(ctx, targetDuration)
    defer cancel()

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            for {
                select {
                case <-testCtx.Done():
                    return
                default:
                    elapsed := time.Since(startTime)
                    rampUpProgress := float64(elapsed) / float64(rampUpDuration)
                    if rampUpProgress > 1.0 {
                        rampUpProgress = 1.0
                    }

                    // Opóźnienie, które maleje wraz z upływem czasu
                    delay := time.Duration(float64(time.Millisecond) * (1 - rampUpProgress) * 100)
                    if delay > 0 {
                        time.Sleep(delay)
                    }

                    // Wykonanie pojedynczych operacji
                    opID := atomic.AddInt64(&ops, 1)

                    // INSERT
                    _, err := pool.Exec(testCtx, "INSERT INTO my_data (id, data) VALUES ($1, $2)", opID, fmt.Sprintf(`{"test": %d}`, opID))
                    if err != nil { atomic.AddInt64(&errors, 1); continue }

                    // UPDATE
                    _, err = pool.Exec(testCtx, "UPDATE my_data SET data = data || $1 WHERE id = $2", fmt.Sprintf(`{"u": %d}`, opID), opID)
                    if err != nil { atomic.AddInt64(&errors, 1); continue }

                    // DELETE
                    _, err = pool.Exec(testCtx, "DELETE FROM my_data WHERE id = $1", opID)
                    if err != nil { atomic.AddInt64(&errors, 1); continue }
                }
            }
        }(i)
    }
    
    wg.Wait()
    duration := time.Since(startTime)

    // Liczba pozostałych wierszy po teście
    var remaining int
    pool.QueryRow(ctx, "SELECT COUNT(*) FROM my_data").Scan(&remaining)

    result := TestResult{
        dbName:     "Postgres",
        duration:   duration,
        operations: ops,
        errors:     errors,
    }
    log.Printf("PostgreSQL: Zakończono test. Czas trwania: %s, Operacji: %d, Błędów: %d, Pozostałych wierszy: %d",
        result.duration.Truncate(time.Second), result.operations, result.errors, remaining)

    return result
}

// === MongoDB ===
func runMongoTest(ctx context.Context) TestResult {
    log.Println("MongoDB: Rozpoczynam test CRUD...")

    clientOpts := options.Client().
        ApplyURI(fmt.Sprintf("mongodb://admin:password123@%s:27017/admin", dbHost)).
        SetMaxPoolSize(256).
        SetMinPoolSize(32).
        SetServerSelectionTimeout(5 * time.Second).
        SetSocketTimeout(10 * time.Second).
        SetConnectTimeout(5 * time.Second)
    client, err := mongo.NewClient(clientOpts)
    if err != nil {
        log.Printf("MongoDB: Błąd klienta: %v", err)
        return TestResult{dbName: "Mongo", errors: -1}
    }
    if err := client.Connect(ctx); err != nil {
        log.Printf("MongoDB: Błąd połączenia: %v", err)
        return TestResult{dbName: "Mongo", errors: -1}
    }
    defer client.Disconnect(ctx)

    // Użycie domyślnego WriteConcern dla maksymalnej wydajności w teście QPS
    collection := client.Database("testdb").Collection("users")
    if _, err := collection.DeleteMany(ctx, bson.M{}); err != nil {
        log.Printf("MongoDB: Błąd oczyszczania: %v", err)
    }

    var ops int64
    var errors int64
    var wg sync.WaitGroup

    startTime := time.Now()
    testCtx, cancel := context.WithTimeout(ctx, targetDuration)
    defer cancel()

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            for {
                select {
                case <-testCtx.Done():
                    return
                default:
                    elapsed := time.Since(startTime)
                    rampUpProgress := float64(elapsed) / float64(rampUpDuration)
                    if rampUpProgress > 1.0 {
                        rampUpProgress = 1.0
                    }
                    
                    delay := time.Duration(float64(time.Millisecond) * (1 - rampUpProgress) * 100)
                    if delay > 0 {
                        time.Sleep(delay)
                    }

                    batchOps := make([]mongo.WriteModel, 0, batchSize*3) // insert + update + delete
                    for j := 0; j < batchSize; j++ {
                        batchID := atomic.AddInt64(&ops, 1)
                        email := fmt.Sprintf("user_%d_%d@test.com", batchID, time.Now().UnixNano())

                        // Tworzenie operacji zapisu (insert, update, delete)
                        batchOps = append(batchOps, &mongo.InsertOneModel{Document: bson.D{{Key: "email", Value: email}, {Key: "name", Value: fmt.Sprintf("User%d", batchID)}}})
                        batchOps = append(batchOps, &mongo.UpdateOneModel{Filter: bson.D{{Key: "email", Value: email}}, Update: bson.D{{Key: "$set", Value: bson.D{{Key: "updated", Value: true}}}}})
                        batchOps = append(batchOps, &mongo.DeleteOneModel{Filter: bson.D{{Key: "email", Value: email}}})
                    }
                    
                    // Użycie BulkWrite; unordered dla maksymalnej przepustowości
                    _, err := collection.BulkWrite(
                        testCtx,
                        batchOps,
                        options.BulkWrite().SetOrdered(false),
                    )
                    if err != nil {
                        atomic.AddInt64(&errors, 1)
                        log.Printf("MongoDB Błąd operacji: %v", err)
                    }
                }
            }
        }(i)
    }
    
    wg.Wait()
    duration := time.Since(startTime)

    remaining, err := collection.CountDocuments(ctx, bson.M{})
    if err != nil {
        log.Printf("MongoDB: Błąd liczenia dokumentów: %v", err)
    }

    result := TestResult{
        dbName:     "Mongo",
        duration:   duration,
        operations: ops,
        errors:     errors,
    }
    log.Printf("MongoDB: Zakończono test. Czas trwania: %s, Operacji: %d, Błędów: %d, Pozostałych dokumentów: %d",
        result.duration.Truncate(time.Second), result.operations, result.errors, remaining)
    return result
}

// === Elasticsearch ===
func runEsTest(ctx context.Context) TestResult {
    log.Println("Elasticsearch: Rozpoczynam test CRUD...")

    es, err := elasticsearch.NewClient(elasticsearch.Config{
        Addresses: []string{fmt.Sprintf("http://%s:9200", dbHost)},
    })
    if err != nil {
        log.Printf("Elasticsearch: Błąd klienta: %v", err)
        return TestResult{dbName: "Elasticsearch", errors: -1}
    }

    indexName := "test_data"
    es.Indices.Delete([]string{indexName}, es.Indices.Delete.WithIgnoreUnavailable(true))
    es.Indices.Create(indexName)
    es.Indices.Refresh()

    var ops int64
    var errors int64
    var wg sync.WaitGroup

    startTime := time.Now()
    testCtx, cancel := context.WithTimeout(ctx, targetDuration)
    defer cancel()

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            for {
                select {
                case <-testCtx.Done():
                    return
                default:
                    elapsed := time.Since(startTime)
                    rampUpProgress := float64(elapsed) / float64(rampUpDuration)
                    if rampUpProgress > 1.0 {
                        rampUpProgress = 1.0
                    }
                    
                    delay := time.Duration(float64(time.Millisecond) * (1 - rampUpProgress) * 100)
                    if delay > 0 {
                        time.Sleep(delay)
                    }

                    var bulkBody string
                    for j := 0; j < batchSize; j++ {
                        batchID := atomic.AddInt64(&ops, 1)
                        docID := fmt.Sprintf("doc-%d-%d", batchID, time.Now().UnixNano())
                        
                        // Wiersze dla operacji _bulk
                        bulkBody += fmt.Sprintf(`{"index": {"_index": "%s", "_id": "%s"}}%s`, indexName, docID, "\n")
                        bulkBody += fmt.Sprintf(`{"value": %d}%s`, batchID, "\n")
                        bulkBody += fmt.Sprintf(`{"update": {"_index": "%s", "_id": "%s"}}%s`, indexName, docID, "\n")
                        bulkBody += fmt.Sprintf(`{"doc": {"updated": true}}%s`, "\n")
                    }

                    // Wysyłanie pakietu operacji _bulk
                    res, err := es.Bulk(
                        strings.NewReader(bulkBody),
                        es.Bulk.WithContext(testCtx),
                    )
                    if err != nil {
                        atomic.AddInt64(&errors, 1)
                        log.Printf("Elasticsearch Błąd operacji: %v", err)
                        continue
                    }
                    // Zawsze zamykaj ciało odpowiedzi, nawet w przypadku błędu.
                    io.Copy(io.Discard, res.Body) // Czytamy i odrzucamy ciało, aby zamknąć połączenie
                    res.Body.Close()
                }
            }
        }(i)
    }

    wg.Wait()
    duration := time.Since(startTime)
    
    countRes, err := es.Count(es.Count.WithIndex(indexName))
    var remaining int64
    if err != nil {
        log.Printf("Elasticsearch: Błąd liczenia dokumentów: %v", err)
    } else {
        countBodyBytes, readErr := io.ReadAll(countRes.Body)
        if readErr != nil {
            log.Printf("Elasticsearch: Błąd odczytu ciała odpowiedzi: %v", readErr)
        } else {
            countBody := make(map[string]interface{})
            if json.Unmarshal(countBodyBytes, &countBody) == nil {
                if count, ok := countBody["count"].(float64); ok {
                    remaining = int64(count)
                }
            }
        }
        countRes.Body.Close()
    }

    result := TestResult{
        dbName:     "Elasticsearch",
        duration:   duration,
        operations: ops,
        errors:     errors,
    }
    log.Printf("Elasticsearch: Zakończono test. Czas trwania: %s, Operacji: %d, Błędów: %d, Pozostałych dokumentów: %d",
        result.duration.Truncate(time.Second), result.operations, result.errors, remaining)
    return result
}

func main() {
    log.Println("🚀 TEST WYDAJNOŚCI CRUD DLA BAZ DANYCH W GO")
    log.Println("======================================")
    log.Printf("Parametry: %s, %d Goroutines na bazę, wielkość pakietu: %d, narastanie przez %s",
        targetDuration, numGoroutines, batchSize, rampUpDuration)
    log.Println("")

    testFuncs := map[string]func(context.Context) TestResult{
        "postgres": runPostgresTest,
        "mongo":    runMongoTest,
        "es":       runEsTest,
    }

    ctx := context.Background()
    resultsChan := make(chan TestResult, len(databasesToTest))
    var wg sync.WaitGroup

    startTime := time.Now()

    for _, dbName := range databasesToTest {
        if testFunc, ok := testFuncs[dbName]; ok {
            wg.Add(1)
            go func(f func(context.Context) TestResult) {
                defer wg.Done()
                resultsChan <- f(ctx)
            }(testFunc)
        }
    }

    wg.Wait()
    close(resultsChan)

    var totalOps int64
    for result := range resultsChan {
        totalOps += result.operations
    }
    
    totalDuration := time.Since(startTime)

    log.Println("\n✅ Podsumowanie testów:")
    log.Printf("Całkowity czas: %s", totalDuration.Truncate(time.Second))
    log.Printf("Intensywność: %.0f operacji/sekundę", float64(totalOps)/totalDuration.Seconds())
    log.Println("📊 Sprawdź szczegółowe metryki w Grafana: http://localhost:3000")
}
