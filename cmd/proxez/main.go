package main

import (
	"log"
	"net/http"

	"highwire/m/internal/backend"
	"highwire/m/internal/proxy"
	"highwire/m/internal/strategy"
)

func main() {

	backends := []*backend.Backend{
		backend.New("alpha", "http://127.0.0.1:9000", 5),
		backend.New("beta", "http://127.0.0.1:9001", 3),
		backend.New("charlie", "http://127.0.0.1:9002", 2),
	}

	strategy := strategy.NewStrategy()

	// Health check all the registered backends
	for i := range backends {
		isHealthy := backends[i].HealthCheck()
		if isHealthy {
			backends[i].MarkHealthy()
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// Implement a backend seletion strategy
		backend, err := strategy.WeightedRoundRobin(backends)

		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		if err := backend.Start(); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		defer func() {
			_ = backend.Complete()
		}()

		// Implement the proxy
		proxy.Proxy(backend, w, r)

	})

	log.Println("proxy listening on http://127.0.0.1:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
