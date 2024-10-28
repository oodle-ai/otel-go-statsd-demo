package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cactus/go-statsd-client/v5/statsd"
)

var (
	statsdClient statsd.Statter

	maxDBConnections     = 1000
	numCustomers         = 100
	numOperations        = 10
	dbPool               = make(chan struct{}, maxDBConnections)
	lastSpikeTime        time.Time
	spikeDuration        = 30 * time.Second
	spikeInterval        = 5 * time.Minute
	spikeMutex           sync.Mutex
	currentSpikeCustomer string
)

func main() {
	var err error
	statsdClient, err = statsd.NewClientWithConfig(&statsd.ClientConfig{
		Address: "otel-collector:8125",
		Prefix:  "go-statsd-demo",
	})
	if err != nil {
		log.Fatalf("Failed to create StatsD client: %v", err)
	}
	defer statsdClient.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "6767"
	}

	statsdClient.Gauge("num_customers", int64(numCustomers), 1.0)
	statsdClient.Gauge("num_operations", int64(numOperations), 1.0)
	go runClient(port)

	http.HandleFunc("/", handleRequest)

	log.Printf("Server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	operation := r.Header.Get("X-Operation")
	if operation == "" {
		operation = "unknown"
	}

	status := http.StatusOK
	customer := r.Header.Get("X-Customer")
	if customer == "" {
		customer = "unknown"
	}

	if !acquireDBConnection(customer) {
		statsdClient.Inc(
			"errors",
			1,
			1.0,
			statsd.Tag{"cause", "db_connection_timeout"},
			statsd.Tag{"customer", customer},
			statsd.Tag{"operation", operation},
		)
		http.Error(w, "Database Connection Timeout", http.StatusServiceUnavailable)
		return
	}
	defer releaseDBConnection()

	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	if rand.Float32() < 0.05 {
		statsdClient.Inc(
			"errors",
			1,
			1.0,
			statsd.Tag{"cause", "internal_server_error"},
			statsd.Tag{"customer", customer},
			statsd.Tag{"operation", operation},
		)
		status = http.StatusInternalServerError
		http.Error(w, "Internal Server Error", status)
	} else {
		fmt.Fprintf(w, "Hello, %s! Operation: %s", customer, operation)
	}

	statsdClient.Inc(
		"requests",
		1,
		1.0,
		statsd.Tag{"operation", operation},
		statsd.Tag{"status", fmt.Sprintf("%d", status)},
		statsd.Tag{"customer", customer},
	)
	statsdClient.TimingDuration(
		"latency",
		time.Since(start),
		1.0,
		statsd.Tag{"operation", operation},
		statsd.Tag{"status", fmt.Sprintf("%d", status)},
		statsd.Tag{"customer", customer},
	)
}

func runClient(port string) {
	for {
		rps := getRequestsPerSecond()
		ticker := time.NewTicker(time.Second / time.Duration(rps))

		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())

		for i := 0; i < rps; i++ {
			select {
			case <-ticker.C:
				wg.Add(1)
				go func() {
					defer wg.Done()
					customer := fmt.Sprintf("customer-%d", rand.Intn(numCustomers))
					operation := fmt.Sprintf("operation-%d", rand.Intn(numOperations))
					req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%s", port), nil)
					req.Header.Set("X-Customer", customer)
					req.Header.Set("X-Operation", operation)
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						log.Printf("Error making request: %v", err)
						return
					}
					defer resp.Body.Close()

					// Discard response body
					_, _ = io.Copy(io.Discard, resp.Body)
				}()
			case <-ctx.Done():
				break
			}
		}

		ticker.Stop()
		wg.Wait()
		cancel()
	}
}

func getRequestsPerSecond() int {
	return 500
}

func acquireDBConnection(customer string) bool {
	start := time.Now()
	delay := calculateDBConnectionDelay(customer)
	time.Sleep(delay)

	select {
	case dbPool <- struct{}{}:
		statsdClient.TimingDuration("db_connection_latency", time.Since(start), 1.0, statsd.Tag{"customer", customer})
		return true
	case <-time.After(50 * time.Millisecond):
		return false
	}
}

func releaseDBConnection() {
	<-dbPool
}

func calculateDBConnectionDelay(customer string) time.Duration {
	spikeMutex.Lock()
	defer spikeMutex.Unlock()

	now := time.Now()
	if now.Sub(lastSpikeTime) > spikeInterval {
		lastSpikeTime = now
		currentSpikeCustomer = fmt.Sprintf("customer-%d", rand.Intn(numCustomers))
	}

	if now.Sub(lastSpikeTime) > spikeDuration {
		currentSpikeCustomer = ""
	}

	if now.Sub(lastSpikeTime) < spikeDuration && customer == currentSpikeCustomer {
		statsdClient.Inc("spike_requests", 1, 1.0, statsd.Tag{"customer", customer})
		return time.Duration(rand.Intn(500)+1500) * time.Millisecond
	}

	return time.Duration(rand.Intn(50)) * time.Millisecond
}
