package systeminfo

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type response struct {
	Name          string    `json:"name"`
	APIVersion    string    `json:"apiVersion"`
	ServerVersion string    `json:"serverVersion"`
	ServerTime    time.Time `json:"serverTime"`
}

func NewHandler(version string, now func() time.Time) http.Handler {
	if now == nil {
		now = time.Now
	}
	router := chi.NewRouter()
	router.Get("/info", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		_ = json.NewEncoder(writer).Encode(response{
			Name: "Flare", APIVersion: "v1", ServerVersion: version, ServerTime: now().UTC(),
		})
	})
	return router
}
