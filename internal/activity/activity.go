package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const activityQuery = `SELECT "Id", "Kind", "Action", "Target", "Timestamp", "Result", "Actor"
FROM (
    SELECT "Id",
           CASE WHEN "Action" LIKE 'auth.%' THEN 'Security' ELSE 'Infrastructure' END AS "Kind",
           "Action", "Target", "Timestamp", "Result", "Actor"
    FROM "AuditEvents"
    UNION ALL
    SELECT "Id", 'Infrastructure' AS "Kind", "Action", "Target", "Timestamp", "Result", NULL::text AS "Actor"
    FROM "InfrastructureEvents"
) AS events
ORDER BY "Timestamp" DESC, "Id" DESC
LIMIT $1 OFFSET $2`

type database interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type Handler struct {
	reader *Reader
	logger *slog.Logger
}

type Reader struct {
	database database
}

type response struct {
	Items    []Event `json:"items"`
	Page     int     `json:"page"`
	PageSize int     `json:"pageSize"`
	HasMore  bool    `json:"hasMore"`
}

type Event struct {
	ID        uuid.UUID `json:"id"`
	Kind      string    `json:"kind"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Timestamp time.Time `json:"timestamp"`
	Result    string    `json:"result"`
	Actor     *string   `json:"actor"`
}

func NewHandler(database database, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{reader: NewReader(database), logger: logger}
}

func NewReader(database database) *Reader {
	return &Reader{database: database}
}

func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writeProblem(writer, http.StatusMethodNotAllowed, "Method not allowed.", "")
		return
	}
	page, pageSize, err := pagination(request)
	if err != nil {
		writeProblem(writer, http.StatusBadRequest, "Invalid request.", "Page and pageSize must be integers in the supported range.")
		return
	}
	items, hasMore, err := handler.reader.Read(request.Context(), page, pageSize)
	if errors.Is(err, context.Canceled) {
		return
	}
	if err != nil {
		handler.logger.Error("Activity query failed", "request_id", middleware.GetReqID(request.Context()), "error", err)
		writeProblem(writer, http.StatusInternalServerError, "Request failed.", "The activity feed could not be loaded.")
		return
	}
	writeJSON(writer, http.StatusOK, response{Items: items, Page: page, PageSize: pageSize, HasMore: hasMore})
}

func (reader *Reader) Read(ctx context.Context, page, pageSize int) ([]Event, bool, error) {
	offset := int64(page-1) * int64(pageSize)
	rows, err := reader.database.Query(ctx, activityQuery, pageSize+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]Event, 0, pageSize)
	for rows.Next() {
		var item Event
		var rawAction string
		var result int
		var actor sql.NullString
		if err := rows.Scan(&item.ID, &item.Kind, &rawAction, &item.Target, &item.Timestamp, &result, &actor); err != nil {
			return nil, false, err
		}
		if actor.Valid {
			item.Actor = &actor.String
		}
		item.Action = displayAction(rawAction)
		item.Result = operationResult(result)
		item.Timestamp = item.Timestamp.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(items) > pageSize
	if hasMore {
		items = items[:pageSize]
	}
	return items, hasMore, nil
}

func pagination(request *http.Request) (page, pageSize int, err error) {
	page, pageSize = 1, 30
	if raw := request.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, err
		}
	}
	if raw := request.URL.Query().Get("pageSize"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, err
		}
	}
	if page < 1 {
		page = 1
	}
	if page > 1_000_000 {
		return 0, 0, errors.New("page exceeds supported range")
	}
	pageSize = min(max(pageSize, 1), 100)
	return page, pageSize, nil
}

func displayAction(action string) string {
	if value, found := map[string]string{
		"auth.bootstrap":               "Administrator bootstrapped",
		"auth.login":                   "Login",
		"auth.logout":                  "Logout",
		"auth.refresh":                 "Session refreshed",
		"container.start":              "Container started",
		"container.stop":               "Container stopped",
		"container.restart":            "Container restarted",
		"container.state.changed":      "Container state changed",
		"container.unexpected_stop":    "Container stopped unexpectedly",
		"container.out_of_memory":      "Container ran out of memory",
		"container.unhealthy":          "Container became unhealthy",
		"container.recovered":          "Container recovered",
		"container.restart_loop":       "Container restart loop detected",
		"coolify.application.start":    "Application started",
		"coolify.application.stop":     "Application stopped",
		"coolify.application.restart":  "Application restarted",
		"coolify.application.redeploy": "Deployment triggered",
		"coolify.service.restart":      "Service restarted",
		"deployment.failed":            "Deployment failed",
		"deployment.status.changed":    "Deployment status changed",
		"server.disconnected":          "Server disconnected",
		"server.reconnected":           "Server reconnected",
	}[action]; found {
		return value
	}
	return action
}

func operationResult(value int) string {
	if value == 0 {
		return "Succeeded"
	}
	return "Failed"
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeProblem(writer http.ResponseWriter, status int, title, detail string) {
	value := map[string]any{"type": "about:blank", "title": title, "status": status}
	if detail != "" {
		value["detail"] = detail
	}
	writer.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
