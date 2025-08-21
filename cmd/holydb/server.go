package holydb

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/garder500/holydb/pkg/storage"
	"github.com/gorilla/mux"
)

func runServer(addr, root string) error {
	ls := &storage.LocalStorage{Root: root}
	r := mux.NewRouter()
	r.Use(loggingMiddleware)
	api := r.PathPrefix("/v1/storage").Subrouter()

	api.HandleFunc("/{bucket}/{rest:.*}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		bucket := vars["bucket"]
		key := vars["rest"]
		switch req.Method {
		case http.MethodPut:
			// support multipart query params
			q := req.URL.Query()
			uploadID := q.Get("uploadId")
			partNum := q.Get("partNumber")
			if uploadID != "" && partNum != "" {
				pn, _ := strconv.Atoi(partNum)
				if err := ls.UploadPart(req.Context(), bucket, key, uploadID, pn, req.Body); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				return
			}
			// normal put with optional metadata header X-Meta-JSON
			var meta storage.Metadata
			if m := req.Header.Get("X-Meta-JSON"); m != "" {
				json.Unmarshal([]byte(m), &meta)
			}
			if err := ls.PutWithMetadata(req.Context(), bucket, key, req.Body, meta); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			// serve object
			rc, err := ls.Get(req.Context(), bucket, key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			defer rc.Close()
			io.Copy(w, rc)
		case http.MethodDelete:
			if err := ls.Delete(req.Context(), bucket, key); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}).Methods(http.MethodPut, http.MethodGet, http.MethodDelete)

	// bucket-level handlers
	api.HandleFunc("/{bucket}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		bucket := vars["bucket"]
		switch req.Method {
		case http.MethodPost:
			// start multipart if ?uploads
			if req.URL.Query().Get("uploads") != "" {
				id, err := ls.StartMultipart(req.Context(), bucket, "")
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write([]byte(id))
				return
			}
			http.Error(w, "unsupported", http.StatusBadRequest)
		case http.MethodGet:
			// list objects with ?prefix=
			prefix := req.URL.Query().Get("prefix")
			list, err := ls.List(req.Context(), bucket, prefix)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			b, _ := json.Marshal(list)
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}).Methods(http.MethodPost, http.MethodGet)

	// bucket metadata
	api.HandleFunc("/{bucket}/.bucket.meta", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		bucket := vars["bucket"]
		switch req.Method {
		case http.MethodPut:
			var bm storage.BucketMetadata
			if err := json.NewDecoder(req.Body).Decode(&bm); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := ls.PutBucketMetadata(req.Context(), bucket, bm); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			bm, err := ls.GetBucketMetadata(req.Context(), bucket)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(bm)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}).Methods(http.MethodPut, http.MethodGet)

	// misc endpoints: stats and reconstruct
	api.HandleFunc("/{bucket}/_stats", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		bucket := vars["bucket"]
		st, err := ls.Stats(req.Context(), bucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(st)
	}).Methods(http.MethodGet)

	api.HandleFunc("/{bucket}/_reconstruct/{rest:.*}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		bucket := vars["bucket"]
		key := vars["rest"]
		out := req.URL.Query().Get("out")
		// include keys comma-separated
		include := req.URL.Query().Get("include")
		includeKeys := []string{}
		if include != "" {
			includeKeys = append(includeKeys, include)
		}
		if out == "" {
			http.Error(w, "missing out param", http.StatusBadRequest)
			return
		}
		if err := ls.Reconstruct(req.Context(), bucket, key, out, includeKeys); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodPost)

	srv := &http.Server{Addr: addr, Handler: r}
	log.Printf("starting server on %s", addr)
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

// add a simple 'serve' subcommand via Execute
func init() {
	// noop: keep simple; runDefault will not call server. Users can call runServer explicitly.
}
