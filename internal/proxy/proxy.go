package proxy

import (
	"io"
	"net/http"
	"net/url"

	"proxez/m/internal/backend"
)

func Proxy(backend *backend.Backend, w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(backend.Address)

	if err != nil {
		http.Error(w, "Invalid backend address", http.StatusInternalServerError)
		return
	}

	// Build the backend url
	target.Path = r.URL.Path
	target.RawPath = r.URL.RawQuery

	req, err := http.NewRequestWithContext(
		r.Context(),
		r.Method,
		target.String(),
		r.Body,
	)

	if err != nil {
		http.Error(w, "Failed to create backend request", http.StatusInternalServerError)
		return
	}

	// Copy request headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		http.Error(w, "Backend request failed", http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()

	// Copy response headers
	for key, value := range resp.Header {
		for _, value := range value {
			w.Header().Add(key, value)
		}
	}

	// Return the backend status code
	w.WriteHeader(resp.StatusCode)

	// Copy the backend response body to the client
	_, _ = io.Copy(w, resp.Body)

}
