package server

import (
	"encoding/json"
	"net/http"

	"github.com/garder500/holydb/pkg/storage"
	"github.com/gorilla/mux"
)

// RegisterMultipartHandlers adds endpoints to complete or abort a multipart upload once parts have been uploaded.
// POST /{bucket}/{key}?complete=uploadId with optional X-Meta-JSON header completes the upload.
// DELETE /{bucket}/{key}?abort=uploadId aborts the upload.
func RegisterMultipartHandlers(r *mux.Router, ls storage.Storage) {
	r.HandleFunc("/{bucket}/{rest:.*}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		bucket := vars["bucket"]
		key := vars["rest"]
		q := req.URL.Query()
		if uploadID := q.Get("complete"); uploadID != "" && req.Method == http.MethodPost {
			var meta storage.Metadata
			if m := req.Header.Get("X-Meta-JSON"); m != "" {
				_ = json.Unmarshal([]byte(m), &meta)
			}
			if err := ls.CompleteMultipart(req.Context(), bucket, key, uploadID, meta); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		if uploadID := q.Get("abort"); uploadID != "" && req.Method == http.MethodDelete {
			if err := ls.AbortMultipart(req.Context(), bucket, key, uploadID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Let other handlers process if not multipart control.
	}).Methods(http.MethodPost, http.MethodDelete)
}
