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

const ES_URL = "http://localhost:9200"
const INDEX_NAME = "productss"

const MAX_CLIENTS = 100
const TEST_DURATION = 180 * time.Second
const RAMP_UP_DURATION = 120 * time.Second
const OPS_PER_BULK = 50
const WRITE_PROBABILITY_PERCENT = 30

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        MAX_CLIENTS,
		MaxIdleConnsPerHost: MAX_CLIENTS,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 10 * time.Second,
}

var wg sync.WaitGroup

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), TEST_DURATION)
	defer cancel()

	log.Printf("Startuję stress test: %d klientów, czas trwania: %s, ramp-up: %s\n",
		MAX_CLIENTS, TEST_DURATION, RAMP_UP_DURATION)
	log.Printf("Cel: %d%% zapisów (_bulk po %d ops), %d%% odczytów (_search)\n",
		WRITE_PROBABILITY_PERCENT, OPS_PER_BULK, 100-WRITE_PROBABILITY_PERCENT)

	rampUpDelay := RAMP_UP_DURATION / time.Duration(MAX_CLIENTS)

	for i := 0; i < MAX_CLIENTS; i++ {
		wg.Add(1)
		go clientWorker(ctx, i)
		time.Sleep(rampUpDelay)
	}

	log.Printf("Wszyscy klienci (%d) uruchomieni. Test potrwa jeszcze przez resztę %s...", MAX_CLIENTS, TEST_DURATION)

	<-ctx.Done()
	log.Println("Czas testu minął. Czekam na zakończenie pracy klientów...")

	wg.Wait()
	log.Println("Wszyscy klienci zatrzymani. Koniec testu.")
}

func clientWorker(ctx context.Context, id int) {
	defer wg.Done()
	log.Printf("Klient %d: START\n", id)

	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

	for {
		select {
		case <-ctx.Done():
			log.Printf("Klient %d: STOP\n", id)
			return
		default:
			if r.Intn(100) < WRITE_PROBABILITY_PERCENT {
				performBulkWrite(id, r)
			} else {
				performSearch(id, r)
			}
			
			time.Sleep(250 * time.Millisecond)
		}
	}
}

func performSearch(workerID int, r *rand.Rand) {
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
}

func performBulkWrite(workerID int, r *rand.Rand) {
	var body strings.Builder

	for i := 0; i < OPS_PER_BULK; i++ {
		docID := r.Intn(50000) + 1
		opType := r.Intn(3)

		switch opType {
		case 0:
			body.WriteString(fmt.Sprintf(`{"index":{"_index":"%s", "_id":"%d"}}`, INDEX_NAME, docID))
			body.WriteRune('\n')
			body.WriteString(generateRandomProduct(r))
			body.WriteRune('\n')
		case 1:
			body.WriteString(fmt.Sprintf(`{"update":{"_index":"%s", "_id":"%d"}}`, INDEX_NAME, docID))
			body.WriteRune('\n')
			body.WriteString(fmt.Sprintf(`{"doc":{"price":%d, "in_stock":%d}}`, r.Intn(2000), r.Intn(50)))
			body.WriteRune('\n')
		case 2:
			body.WriteString(fmt.Sprintf(`{"delete":{"_index":"%s", "_id":"%d"}}`, INDEX_NAME, docID))
			body.WriteRune('\n')
		}
	}

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
	}
}

func generateRandomProduct(r *rand.Rand) string {
	name := "Testowy Produkt " + strconv.Itoa(r.Intn(10000))
	price := r.Intn(1000)
	inStock := r.Intn(100)
	return fmt.Sprintf(`{"name":"%s", "price":%d, "in_stock":%d, "created_at":"%s"}`,
		name, price, inStock, time.Now().Format(time.RFC3339))
}