package metrics

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

// StartServer démarre un serveur HTTP simple exposant les métriques.
func StartServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(Default.Snapshot()); err != nil {
			log.Printf("écriture métriques impossible: %v", err)
		}
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("serveur métriques arrêté: %v", err)
		}
	}()

	return server
}

// Shutdown arrête proprement un serveur de métriques.
func Shutdown(ctx context.Context, server *http.Server) {
	if server == nil {
		return
	}
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("arrêt serveur métriques impossible: %v", err)
	}
}
