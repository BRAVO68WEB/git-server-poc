package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"githut/internal/config"
	"githut/internal/database"
	"githut/internal/git"
	"githut/internal/lfs"
	"githut/internal/observability"
)

func main() {
	addr := os.Getenv("GITHUT_HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	mux := http.NewServeMux()
	cfg := config.Load()
	git.RegisterHTTP(mux, cfg)
	observability.RegisterMetrics(mux)
	lfs.RegisterHTTP(mux, cfg)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	log.Printf("serve listening on %s", addr)
	if cfg.SSHAddr != "" && cfg.PostgresDSN != "" {
		go func() {
			db, err := database.Connect(context.Background(), cfg.PostgresDSN)
			if err != nil {
				log.Printf("ssh start error: %v", err)
				return
			}
			defer db.Close(context.Background())
			if err := git.StartSSH(context.Background(), cfg.SSHAddr, db); err != nil {
				log.Printf("ssh start error: %v", err)
			}
		}()
	}
	err := http.ListenAndServe(addr, logMiddleware(mux))
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingResponseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(lw, r)
		d := time.Since(start)
		log.Printf(`{"ts":"%s","method":"%s","path":"%s","status":%d,"duration_ms":%d}`, time.Now().Format(time.RFC3339), r.Method, r.URL.Path, lw.status, d.Milliseconds())
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.status = code
	lw.ResponseWriter.WriteHeader(code)
}
