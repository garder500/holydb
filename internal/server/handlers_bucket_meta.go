package server

import (
	"encoding/json"
	"net/http"

	"github.com/garder500/holydb/pkg/storage"
	"github.com/gorilla/mux"
)

// RegisterBucketMetaHandlers handles bucket metadata read/write (.bucket.meta)
func RegisterBucketMetaHandlers(r *mux.Router, ls storage.Storage) {
	r.HandleFunc("/{bucket}/.bucket.meta", func(w http.ResponseWriter, req *http.Request) {
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
}
