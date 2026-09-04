package httpapi

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

//go:embed public.html assets
var publicContent embed.FS

var publicTemplate = template.Must(template.ParseFS(publicContent, "public.html"))

type publicPageData struct {
	Version        string
	Uptime         string
	GeneratedAt    string
	DatabaseStatus string
	DatabaseClass  string
}

func registerPublicRoutes(router chi.Router, version string, database databasePinger, startedAt time.Time) {
	if strings.TrimSpace(version) == "" {
		version = "Unavailable"
	}
	assets, err := fs.Sub(publicContent, "assets")
	if err != nil {
		panic(err)
	}
	assetHandler := http.StripPrefix("/assets/", http.FileServer(http.FS(assets)))
	router.Handle("/assets/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		assetHandler.ServeHTTP(w, r)
	}))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		databaseStatus, databaseClass := dependencyStatus(r.Context(), database)
		data := publicPageData{
			Version:        version,
			Uptime:         formatUptime(time.Since(startedAt)),
			GeneratedAt:    time.Now().UTC().Format("2 Jan 2006, 15:04 UTC"),
			DatabaseStatus: databaseStatus,
			DatabaseClass:  databaseClass,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; font-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if err := publicTemplate.Execute(w, data); err != nil {
			return
		}
	})
}

func dependencyStatus(parent context.Context, database databasePinger) (string, string) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	if err := database.Ping(ctx); err != nil {
		return "Unavailable", "unavailable"
	}
	return "Operational", "operational"
}

func formatUptime(value time.Duration) string {
	if value < time.Minute {
		return "Less than a minute"
	}
	value = value.Truncate(time.Minute)
	days := int(value / (24 * time.Hour))
	value %= 24 * time.Hour
	hours := int(value / time.Hour)
	minutes := int(value%time.Hour) / int(time.Minute)
	if days > 0 {
		return strings.TrimSpace(strings.Join([]string{plural(days, "day"), plural(hours, "hour")}, " "))
	}
	if hours > 0 {
		return strings.TrimSpace(strings.Join([]string{plural(hours, "hour"), plural(minutes, "minute")}, " "))
	}
	return plural(minutes, "minute")
}

func plural(value int, unit string) string {
	if value == 0 {
		return ""
	}
	if value != 1 {
		unit += "s"
	}
	return strconv.Itoa(value) + " " + unit
}
