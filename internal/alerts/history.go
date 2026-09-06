package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/itsmangooo/flare/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const listAlertsQuery = `SELECT a."Id",a."Kind",a."Severity",a."Title",a."Message",a."Source",
       a."ResourceType",a."ResourceId",a."Status",a."FirstSeenAt",a."LastSeenAt",
       a."RecoveredAt",a."OccurrenceCount",r."ReadAt"
FROM "Alerts" a
LEFT JOIN "AlertReads" r ON r."AlertId"=a."Id" AND r."UserId"=$1
WHERE (NOT $2 OR r."ReadAt" IS NULL)
ORDER BY a."LastSeenAt" DESC,a."Id" DESC
LIMIT $3 OFFSET $4`

const unreadCountQuery = `SELECT count(*)
FROM "Alerts" a
LEFT JOIN "AlertReads" r ON r."AlertId"=a."Id" AND r."UserId"=$1
WHERE r."ReadAt" IS NULL`

type historyDatabase interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type HistoryHandler struct {
	database    historyDatabase
	logger      *slog.Logger
	now         func() time.Time
	currentUser func(context.Context) (auth.User, bool)
	hasRole     func(context.Context, string) bool
	preferences *PreferenceStore
	router      http.Handler
}

type Alert struct {
	ID              uuid.UUID  `json:"id"`
	Kind            string     `json:"kind"`
	Severity        string     `json:"severity"`
	Title           string     `json:"title"`
	Message         string     `json:"message"`
	Source          string     `json:"source"`
	ResourceType    *string    `json:"resourceType"`
	ResourceID      *string    `json:"resourceId"`
	Status          string     `json:"status"`
	FirstSeenAt     time.Time  `json:"firstSeenAt"`
	LastSeenAt      time.Time  `json:"lastSeenAt"`
	RecoveredAt     *time.Time `json:"recoveredAt"`
	OccurrenceCount int        `json:"occurrenceCount"`
	ReadAt          *time.Time `json:"readAt"`
}

type alertHistoryResponse struct {
	Items       []Alert `json:"items"`
	Page        int     `json:"page"`
	PageSize    int     `json:"pageSize"`
	HasMore     bool    `json:"hasMore"`
	UnreadCount int64   `json:"unreadCount"`
}

func NewHistoryHandler(database historyDatabase, logger *slog.Logger) *HistoryHandler {
	if logger == nil {
		logger = slog.Default()
	}
	handler := &HistoryHandler{
		database: database, logger: logger, now: time.Now, currentUser: auth.UserFromContext,
		hasRole: auth.HasRole, preferences: NewPreferenceStore(database),
	}
	router := chi.NewRouter()
	router.Get("/", handler.list)
	router.Get("/preferences", handler.getPreferences)
	router.Put("/preferences", handler.putPreferences)
	router.Put("/{id}/read", handler.markRead)
	router.Delete("/{id}/read", handler.markUnread)
	handler.router = router
	return handler
}

func (handler *HistoryHandler) getPreferences(writer http.ResponseWriter, request *http.Request) {
	if _, ok := handler.currentUser(request.Context()); !ok {
		writeHistoryProblem(writer, http.StatusUnauthorized, "Unauthorized.", "")
		return
	}
	preferences, err := handler.preferences.Get(request.Context())
	if err != nil {
		handler.failure(writer, request, "Notification preferences query failed", err)
		return
	}
	writeHistoryJSON(writer, http.StatusOK, preferences)
}

func (handler *HistoryHandler) putPreferences(writer http.ResponseWriter, request *http.Request) {
	user, ok := handler.currentUser(request.Context())
	if !ok {
		writeHistoryProblem(writer, http.StatusUnauthorized, "Unauthorized.", "")
		return
	}
	if !handler.hasRole(request.Context(), "Administrator") {
		writeHistoryProblem(writer, http.StatusForbidden, "Forbidden.", "Administrator access is required.")
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, 8<<10)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var preferences Preferences
	if err := decoder.Decode(&preferences); err != nil {
		writeHistoryProblem(writer, http.StatusBadRequest, "Invalid request.", "Notification preferences are invalid.")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeHistoryProblem(writer, http.StatusBadRequest, "Invalid request.", "Only one JSON object is allowed.")
		return
	}
	preferences.MinimumSeverity = strings.ToLower(strings.TrimSpace(preferences.MinimumSeverity))
	if err := validatePreferences(preferences); err != nil {
		writeHistoryProblem(writer, http.StatusBadRequest, "Invalid request.", err.Error())
		return
	}
	_, err := handler.database.Exec(request.Context(), `INSERT INTO "NotificationPreferences"
    ("Id","Enabled","MinimumSeverity","RecoveryEnabled","DockerEnabled","CoolifyEnabled","CloudflareEnabled","HostEnabled","UpdatedAt","UpdatedBy")
VALUES (1,$1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT ("Id") DO UPDATE SET "Enabled"=EXCLUDED."Enabled","MinimumSeverity"=EXCLUDED."MinimumSeverity",
    "RecoveryEnabled"=EXCLUDED."RecoveryEnabled","DockerEnabled"=EXCLUDED."DockerEnabled",
    "CoolifyEnabled"=EXCLUDED."CoolifyEnabled","CloudflareEnabled"=EXCLUDED."CloudflareEnabled",
    "HostEnabled"=EXCLUDED."HostEnabled","UpdatedAt"=EXCLUDED."UpdatedAt","UpdatedBy"=EXCLUDED."UpdatedBy"`,
		preferences.Enabled, preferences.MinimumSeverity, preferences.RecoveryEnabled, preferences.DockerEnabled,
		preferences.CoolifyEnabled, preferences.CloudflareEnabled, preferences.HostEnabled, handler.now().UTC(), user.ID)
	if err != nil {
		handler.failure(writer, request, "Notification preferences update failed", err)
		return
	}
	writeHistoryJSON(writer, http.StatusOK, preferences)
}

func (handler *HistoryHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler.router.ServeHTTP(writer, request)
}

func (handler *HistoryHandler) list(writer http.ResponseWriter, request *http.Request) {
	user, ok := handler.currentUser(request.Context())
	if !ok {
		writeHistoryProblem(writer, http.StatusUnauthorized, "Unauthorized.", "")
		return
	}
	page, pageSize, unreadOnly, err := alertPagination(request)
	if err != nil {
		writeHistoryProblem(writer, http.StatusBadRequest, "Invalid request.", "Pagination or unreadOnly is invalid.")
		return
	}
	items, hasMore, err := handler.read(request.Context(), user.ID, page, pageSize, unreadOnly)
	if errors.Is(err, context.Canceled) {
		return
	}
	if err != nil {
		handler.failure(writer, request, "Alert history query failed", err)
		return
	}
	var unreadCount int64
	if err = handler.database.QueryRow(request.Context(), unreadCountQuery, user.ID).Scan(&unreadCount); err != nil {
		handler.failure(writer, request, "Unread alert count failed", err)
		return
	}
	writeHistoryJSON(writer, http.StatusOK, alertHistoryResponse{
		Items: items, Page: page, PageSize: pageSize, HasMore: hasMore, UnreadCount: unreadCount,
	})
}

func (handler *HistoryHandler) read(ctx context.Context, userID uuid.UUID, page, pageSize int, unreadOnly bool) ([]Alert, bool, error) {
	offset := int64(page-1) * int64(pageSize)
	rows, err := handler.database.Query(ctx, listAlertsQuery, userID, unreadOnly, pageSize+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]Alert, 0, pageSize)
	for rows.Next() {
		var item Alert
		var resourceType, resourceID sql.NullString
		var recoveredAt, readAt sql.NullTime
		if err = rows.Scan(&item.ID, &item.Kind, &item.Severity, &item.Title, &item.Message, &item.Source,
			&resourceType, &resourceID, &item.Status, &item.FirstSeenAt, &item.LastSeenAt,
			&recoveredAt, &item.OccurrenceCount, &readAt); err != nil {
			return nil, false, err
		}
		if resourceType.Valid {
			item.ResourceType = &resourceType.String
		}
		if resourceID.Valid {
			item.ResourceID = &resourceID.String
		}
		item.FirstSeenAt = item.FirstSeenAt.UTC()
		item.LastSeenAt = item.LastSeenAt.UTC()
		if recoveredAt.Valid {
			value := recoveredAt.Time.UTC()
			item.RecoveredAt = &value
		}
		if readAt.Valid {
			value := readAt.Time.UTC()
			item.ReadAt = &value
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(items) > pageSize
	if hasMore {
		items = items[:pageSize]
	}
	return items, hasMore, nil
}

func (handler *HistoryHandler) markRead(writer http.ResponseWriter, request *http.Request) {
	user, id, ok := handler.identityAndAlertID(writer, request)
	if !ok {
		return
	}
	result, err := handler.database.Exec(request.Context(), `INSERT INTO "AlertReads" ("AlertId","UserId","ReadAt")
SELECT "Id",$2,$3 FROM "Alerts" WHERE "Id"=$1
ON CONFLICT ("AlertId","UserId") DO UPDATE SET "ReadAt"=EXCLUDED."ReadAt"`, id, user.ID, handler.now().UTC())
	if err != nil {
		handler.failure(writer, request, "Mark alert read failed", err)
		return
	}
	if result.RowsAffected() == 0 {
		writeHistoryProblem(writer, http.StatusNotFound, "Alert not found.", "")
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *HistoryHandler) markUnread(writer http.ResponseWriter, request *http.Request) {
	user, id, ok := handler.identityAndAlertID(writer, request)
	if !ok {
		return
	}
	if _, err := handler.database.Exec(request.Context(), `DELETE FROM "AlertReads" WHERE "AlertId"=$1 AND "UserId"=$2`, id, user.ID); err != nil {
		handler.failure(writer, request, "Mark alert unread failed", err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *HistoryHandler) identityAndAlertID(writer http.ResponseWriter, request *http.Request) (auth.User, uuid.UUID, bool) {
	user, ok := handler.currentUser(request.Context())
	if !ok {
		writeHistoryProblem(writer, http.StatusUnauthorized, "Unauthorized.", "")
		return auth.User{}, uuid.Nil, false
	}
	id, err := uuid.Parse(chi.URLParam(request, "id"))
	if err != nil {
		writeHistoryProblem(writer, http.StatusBadRequest, "Invalid request.", "Alert identifier is invalid.")
		return auth.User{}, uuid.Nil, false
	}
	return user, id, true
}

func (handler *HistoryHandler) failure(writer http.ResponseWriter, request *http.Request, message string, err error) {
	handler.logger.Error(message, "request_id", middleware.GetReqID(request.Context()), "error", err)
	writeHistoryProblem(writer, http.StatusInternalServerError, "Request failed.", "Alert history could not be updated.")
}

func alertPagination(request *http.Request) (page, pageSize int, unreadOnly bool, err error) {
	page, pageSize = 1, 30
	if raw := request.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil {
			return
		}
	}
	if raw := request.URL.Query().Get("pageSize"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil {
			return
		}
	}
	if raw := request.URL.Query().Get("unreadOnly"); raw != "" {
		unreadOnly, err = strconv.ParseBool(raw)
		if err != nil {
			return
		}
	}
	if page < 1 {
		page = 1
	}
	if page > 1_000_000 {
		err = errors.New("page exceeds supported range")
		return
	}
	pageSize = min(max(pageSize, 1), 100)
	return
}

func writeHistoryJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeHistoryProblem(writer http.ResponseWriter, status int, title, detail string) {
	value := map[string]any{"type": "about:blank", "title": title, "status": status}
	if detail != "" {
		value["detail"] = detail
	}
	writer.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
