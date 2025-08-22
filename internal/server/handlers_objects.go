package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/garder500/holydb/pkg/storage"
	"github.com/gorilla/mux"
)

// RegisterObjectHandlers registers handlers for object-level operations: PUT/GET/DELETE /{bucket}/{key...}
func RegisterObjectHandlers(r *mux.Router, ls storage.Storage) {
	r.HandleFunc("/{bucket}/{rest:.*}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		bucket := vars["bucket"]
		key := vars["rest"]
		switch req.Method {
		case http.MethodPut:
			q := req.URL.Query()
			uploadID := q.Get("uploadId")
			partNum := q.Get("partNumber")
			if uploadID != "" && partNum != "" { // multipart part upload
				pn, _ := strconv.Atoi(partNum)
				if err := ls.UploadPart(req.Context(), bucket, key, uploadID, pn, req.Body); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				return
			}
			var meta storage.Metadata
			if m := req.Header.Get("X-Meta-JSON"); m != "" {
				_ = json.Unmarshal([]byte(m), &meta)
			}
			if err := ls.PutWithMetadata(req.Context(), bucket, key, req.Body, meta); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
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
}
