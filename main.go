package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- Konfiguracja Testu ---

// Adres serwera Elasticsearch
const ES_URL = "http://localhost:9200"
const INDEX_NAME = "productss"

// Docelowa liczba klientów (goroutin)
const MAX_CLIENTS = 100
// Czas trwania testu (np. "1m", "30s")
const TEST_DURATION = 180 * time.Second
// Czas, w którym wszyscy klienci mają zostać uruchomieni (ramp-up)
const RAMP_UP_DURATION = 120 * time.Second
// Ile operacji (Create, Update, Delete) ma być w jednym żądaniu _bulk
const OPS_PER_BULK = 50
// Jakie jest prawdopodobieństwo (w %), że klient wykona operację zapisu (bulk)
// Reszta to operacje odczytu (search)
const WRITE_PROBABILITY_PERCENT = 30

// --- Koniec Konfiguracji ---

// Globalny klient HTTP, aby ponownie używać połączeń (keep-alive)
// To jest kluczowe dla wydajności!
var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        MAX_CLIENTS,
		MaxIdleConnsPerHost: MAX_CLIENTS,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 10 * time.Second,
}

// Globalna WaitGroup do śledzenia aktywnych goroutin
var wg sync.WaitGroup

func main() {
	// Używamy kontekstu do globalnego zatrzymania wszystkich goroutin po TEST_DURATION
	ctx, cancel := context.WithTimeout(context.Background(), TEST_DURATION)
	defer cancel()

	log.Printf("Startuję stress test: %d klientów, czas trwania: %s, ramp-up: %s\n",
		MAX_CLIENTS, TEST_DURATION, RAMP_UP_DURATION)
	log.Printf("Cel: %d%% zapisów (_bulk po %d ops), %d%% odczytów (_search)\n",
		WRITE_PROBABILITY_PERCENT, OPS_PER_BULK, 100-WRITE_PROBABILITY_PERCENT)

	// Obliczamy opóźnienie między startem kolejnych klientów
	rampUpDelay := RAMP_UP_DURATION / time.Duration(MAX_CLIENTS)

	// Pętla "Ramp-up"
	for i := 0; i < MAX_CLIENTS; i++ {
		wg.Add(1)
		go clientWorker(ctx, i)
		time.Sleep(rampUpDelay)
	}

	log.Printf("Wszyscy klienci (%d) uruchomieni. Test potrwa jeszcze przez resztę %s...", MAX_CLIENTS, TEST_DURATION)

	// Czekaj na sygnał zakończenia (albo przez timeout kontekstu, albo przerwanie)
	<-ctx.Done()
	log.Println("Czas testu minął. Czekam na zakończenie pracy klientów...")

	// Czekaj aż wszystkie goroutiny (które odebrały sygnał z ctx) zakończą się
	wg.Wait()
	log.Println("Wszyscy klienci zatrzymani. Koniec testu.")
}

// clientWorker symuluje jednego klienta
func clientWorker(ctx context.Context, id int) {
	defer wg.Done()
	log.Printf("Klient %d: START\n", id)

	// Każdy klient ma swój własny generator liczb losowych
	// (aby uniknąć globalnej blokady na rand)
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

	// Pętla główna klienta. Działa tak szybko, jak to możliwe.
	for {
		select {
		case <-ctx.Done():
			// Otrzymano sygnał zakończenia testu
			log.Printf("Klient %d: STOP\n", id)
			return
		default:
			// Wykonaj pracę
			if r.Intn(100) < WRITE_PROBABILITY_PERCENT {
				// Wykonaj ZAPIS (Create/Update/Delete) przez _bulk
				performBulkWrite(id, r)
			} else {
				// Wykonaj ODCZYT (Search)
				performSearch(id, r)
			}
			
			// Opcjonalne: małe opóźnienie między operacjami
			time.Sleep(250 * time.Millisecond)
		}
	}
}

// performSearch wykonuje operację odczytu
func performSearch(workerID int, r *rand.Rand) {
	// Proste wyszukiwanie. W realnym teście losuj terminy.
	term := "produkt"
	url := fmt.Sprintf("%s/%s/_search?q=name:%s", ES_URL, INDEX_NAME, term)

	req, _ := http.NewRequest("GET", url, nil)
	resp, err := httpClient.Do(req)

	if err != nil {
		log.Printf("Klient %d [SEARCH]: Błąd żądania: %v\n", workerID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Klient %d [SEARCH]: Błąd statusu: %s\n", workerID, resp.Status)
	}
	// W realnym teście warto by było czytać body, aby symulować pełne użycie
	// io.Copy(io.Discard, resp.Body)
}

// performBulkWrite generuje i wysyła paczkę operacji C/U/D
func performBulkWrite(workerID int, r *rand.Rand) {
	// strings.Builder jest znacznie wydajniejszy niż konkatenacja stringów
	var body strings.Builder

	for i := 0; i < OPS_PER_BULK; i++ {
		// Losujemy ID dokumentu do operacji
		docID := r.Intn(50000) + 1 // ID od 1 do 50000
		opType := r.Intn(3)      // 0 = Create, 1 = Update, 2 = Delete

		switch opType {
		case 0: // CREATE (technicznie 'index')
			// Linia akcji
			body.WriteString(fmt.Sprintf(`{"index":{"_index":"%s", "_id":"%d"}}`, INDEX_NAME, docID))
			body.WriteRune('\n')
			// Linia danych
			body.WriteString(generateRandomProduct(r))
			body.WriteRune('\n')
		case 1: // UPDATE
			// Linia akcji
			body.WriteString(fmt.Sprintf(`{"update":{"_index":"%s", "_id":"%d"}}`, INDEX_NAME, docID))
			body.WriteRune('\n')
			// Linia danych (z 'doc' aby zrobić partial update)
			body.WriteString(fmt.Sprintf(`{"doc":{"price":%d, "in_stock":%d}}`, r.Intn(2000), r.Intn(50)))
			body.WriteRune('\n')
		case 2: // DELETE
			// Linia akcji
			body.WriteString(fmt.Sprintf(`{"delete":{"_index":"%s", "_id":"%d"}}`, INDEX_NAME, docID))
			body.WriteRune('\n')
		}
	}

	// Wyślij żądanie _bulk
	url := fmt.Sprintf("%s/_bulk", ES_URL)
	req, _ := http.NewRequest("POST", url, strings.NewReader(body.String()))
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Klient %d [BULK]: Błąd żądania: %v\n", workerID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Klient %d [BULK]: Błąd statusu: %s\n", workerID, resp.Status)
		// Tutaj warto byłoby odczytać body, aby zobaczyć błędy Elasticsearch
	}
}

// generateRandomProduct tworzy JSON dla nowego produktu
func generateRandomProduct(r *rand.Rand) string {
	name := "Testowy Produkt " + strconv.Itoa(r.Intn(10000))
	price := r.Intn(1000)
	inStock := r.Intn(100)
	return fmt.Sprintf(`{"name":"%s", "price":%d, "in_stock":%d, "created_at":"%s"}`,
		name, price, inStock, time.Now().Format(time.RFC3339))
}