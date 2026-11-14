package main

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"proxez/m/web"

	"proxez/m/internal/backend"
	"proxez/m/internal/proxy"
	"proxez/m/internal/strategy"
)

func main() {

	backends := []*backend.Backend{
		backend.New("alpha", "http://127.0.0.1:9000", 5),
		backend.New("beta", "http://127.0.0.1:9001", 3),
		backend.New("charlie", "http://127.0.0.1:9002", 2),
	}

	stratSelection := strategy.NewStrategy()

	// Health check all the registered backends
	for i := range backends {
		isHealthy := backends[i].HealthCheck()
		if isHealthy {
			backends[i].MarkHealthy()
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// Implement a backend seletion strategy
		selectedBackend, err := stratSelection.WeightedRoundRobin(backends)

		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		if err := selectedBackend.Start(); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		defer func() {
			_ = selectedBackend.Complete()
		}()

		// Implement the proxy
		proxy.Proxy(selectedBackend, w, r)

	})

	// parse the admin page at startup
	tmpl := template.Must(template.ParseFS(web.Files, "templates/index.html"))

	staticFS, err := fs.Sub(web.Files, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]string{"Title": "Load balancer admin"}
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	log.Println("proxy listening on http://127.0.0.1:8080")
	log.Println("admin page on http://localhost:8080/admin")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
