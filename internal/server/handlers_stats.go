package server

import (
	"encoding/json"
	"net/http"

	"github.com/garder500/holydb/pkg/storage"
	"github.com/gorilla/mux"
)

// RegisterStatsHandlers registers bucket stats endpoint /{bucket}/_stats
func RegisterStatsHandlers(r *mux.Router, ls storage.Storage) {
	r.HandleFunc("/{bucket}/_stats", func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		bucket := vars["bucket"]
		st, err := ls.Stats(req.Context(), bucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(st)
	}).Methods(http.MethodGet)
}
