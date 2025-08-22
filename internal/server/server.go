package server

import (
	"log"
	"net/http"

	"github.com/garder500/holydb/pkg/storage"
	"github.com/gorilla/mux"
)

// Config holds server configuration.
type Config struct {
	Addr string
	Root string
}

// New creates a configured *http.Server with all routes registered.
func New(cfg Config) *http.Server {
	ls := &storage.LocalStorage{Root: cfg.Root}
	r := mux.NewRouter()
	r.Use(loggingMiddleware)
	// versioned API root
	v1 := r.PathPrefix("/v1").Subrouter()
	// storage subrouter
	storageRouter := v1.PathPrefix("/storage").Subrouter()

	RegisterObjectHandlers(storageRouter, ls)
	RegisterBucketHandlers(storageRouter, ls)
	RegisterBucketMetaHandlers(storageRouter, ls)
	RegisterStatsHandlers(storageRouter, ls)
	RegisterReconstructHandlers(storageRouter, ls)
	RegisterMultipartHandlers(storageRouter, ls)

	return &http.Server{Addr: cfg.Addr, Handler: r}
}

// Run starts the HTTP server and blocks.
func Run(cfg Config) error {
	srv := New(cfg)
	log.Printf("starting server on %s", cfg.Addr)
	return srv.ListenAndServe()
}

// logging middleware to trace requests and response status
type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (l *loggingResponseWriter) WriteHeader(code int) {
	l.status = code
	l.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := &loggingResponseWriter{ResponseWriter: w, status: 200}
		log.Printf("REQ %s %s from %s", r.Method, r.URL.String(), r.RemoteAddr)
		next.ServeHTTP(lrw, r)
		log.Printf("RES %d %s %s", lrw.status, r.Method, r.URL.Path)
	})
}
