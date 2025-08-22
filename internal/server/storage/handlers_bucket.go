package server

import (
	"encoding/json"
	"net/http"

	"github.com/garder500/holydb/pkg/storage"
	"github.com/gorilla/mux"
)

// RegisterBucketHandlers registers bucket listing and control endpoints (GET list, POST initiate multipart)
func RegisterBucketHandlers(r *mux.Router, ls storage.Storage) {
	r.HandleFunc("/{bucket}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		bucket := vars["bucket"]
		switch req.Method {
		case http.MethodPost:
			if req.URL.Query().Get("uploads") != "" { // start multipart (without key yet)
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
}
