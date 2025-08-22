package server

import (
	"net/http"
	"strings"

	"github.com/garder500/holydb/pkg/storage"
	"github.com/gorilla/mux"
)

// RegisterReconstructHandlers registers /{bucket}/_reconstruct/{key...}
func RegisterReconstructHandlers(r *mux.Router, ls storage.Storage) {
	r.HandleFunc("/{bucket}/_reconstruct/{rest:.*}", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		bucket := vars["bucket"]
		key := vars["rest"]
		out := req.URL.Query().Get("out")
		include := req.URL.Query().Get("include")
		var includeKeys []string
		if include != "" {
			includeKeys = strings.Split(include, ",")
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
}
