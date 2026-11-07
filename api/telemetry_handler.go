// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"golang.org/x/sync/singleflight"
	"google.com/apf/api/core"
	"google.com/apf/api/db"
	"gorm.io/gorm"
)

type APFErrorPayload struct {
	Error APFErrorDetail `json:"error"`
}

type APFErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func sendAPFError(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APFErrorPayload{
		Error: APFErrorDetail{
			Code:    code,
			Message: message,
			Status:  statusCode,
		},
	})
}

type cachedTelemetryItem struct {
	response  *db.TelemetryUsageResponse
	csvData   []byte
	createdAt time.Time
}

// TelemetryHandler serves GET /api/v1/telemetry/usage API endpoint.
type TelemetryHandler struct {
	telemetryStore  *db.ADKTelemetryStore
	orgStore        db.OrganizationStore
	firestoreClient *firestore.Client
	db              *gorm.DB
	cache           map[string]*cachedTelemetryItem
	cacheMu         sync.RWMutex
	sfg             singleflight.Group
	ttl             time.Duration
}

// NewTelemetryHandler creates a new TelemetryHandler instance.
func NewTelemetryHandler(dbConn *gorm.DB, firestoreClient *firestore.Client, orgStore db.OrganizationStore) *TelemetryHandler {
	var store *db.ADKTelemetryStore
	if dbConn != nil {
		store = db.NewADKTelemetryStore(dbConn, firestoreClient)
	}
	return &TelemetryHandler{
		telemetryStore:  store,
		orgStore:        orgStore,
		firestoreClient: firestoreClient,
		db:              dbConn,
		cache:           make(map[string]*cachedTelemetryItem),
		ttl:             5 * time.Minute,
	}
}

// GetTelemetryUsage handles GET /api/v1/telemetry/usage requests.
func (h *TelemetryHandler) GetTelemetryUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		preflightHandler(w, r)
		return
	}

	claims, ok := r.Context().Value(core.UserClaimsKey).(*core.UserClaims)
	if !ok || claims == nil {
		sendAPFError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user authentication context")
		return
	}

	// Strict RBAC: Only Super Admin can access AI usage & telemetry metrics
	if !claims.IsSuperAdmin {
		sendAPFError(w, http.StatusForbidden, "FORBIDDEN", "telemetry access requires Super Admin role")
		return
	}

	q := r.URL.Query()
	reqOrgID := q.Get("organizationId")
	reqProjectID := q.Get("projectId")
	reqUserID := q.Get("userId")
	reqSessionID := q.Get("sessionId")
	startDateStr := q.Get("startDate")
	endDateStr := q.Get("endDate")
	limitStr := q.Get("limit")
	offsetStr := q.Get("offset")
	format := q.Get("format")
	refresh := q.Get("refresh") == "true"

	// Org Existence Check if organizationId is specified
	if reqOrgID != "" && h.orgStore != nil {
		_, err := h.orgStore.GetOrganization(r.Context(), reqOrgID)
		if err != nil {
			sendAPFError(w, http.StatusNotFound, "ORG_NOT_FOUND", fmt.Sprintf("organization %s not found", reqOrgID))
			return
		}
	}

	// Parse date window
	var endDate time.Time
	if endDateStr != "" {
		if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = t
		} else if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
		} else {
			sendAPFError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid endDate format")
			return
		}
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	var startDate time.Time
	if startDateStr != "" {
		if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = t
		} else if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = t
		} else {
			sendAPFError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid startDate format")
			return
		}
	}
	if startDate.IsZero() {
		startDate = endDate.AddDate(0, 0, -30)
	}

	if startDate.After(endDate) {
		sendAPFError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "startDate cannot be after endDate")
		return
	}

	// Pagination params
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 500 {
		log.Printf("WARNING: Requested limit %d exceeds max cap 500, capping to 500", limit)
		limit = 500
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	cacheKey := fmt.Sprintf("org:%s:proj:%s:user:%s:sess:%s:start:%s:end:%s:lim:%d:off:%d",
		reqOrgID, reqProjectID, reqUserID, reqSessionID,
		startDate.Truncate(time.Minute).Format(time.RFC3339),
		endDate.Truncate(time.Minute).Format(time.RFC3339),
		limit, offset)

	// Check cache if not forcing refresh
	if !refresh {
		h.cacheMu.RLock()
		item, cached := h.cache[cacheKey]
		h.cacheMu.RUnlock()

		if cached && time.Since(item.createdAt) < h.ttl {
			if format == "csv" {
				fn := "telemetry-usage.csv"
				if reqOrgID != "" {
					fn = fmt.Sprintf("telemetry-%s.csv", reqOrgID)
				} else if reqSessionID != "" {
					fn = fmt.Sprintf("telemetry-%s.csv", reqSessionID)
				}
				w.Header().Set("Content-Type", "text/csv")
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fn))
				w.WriteHeader(http.StatusOK)
				w.Write(item.csvData)
				return
			}
			sendJSON(w, http.StatusOK, item.response)
			return
		}
	}

	sfKey := cacheKey
	if refresh {
		sfKey = cacheKey + ":refresh:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	// Coalesce concurrent queries via singleflight
	res, err, _ := h.sfg.Do(sfKey, func() (interface{}, error) {
		if h.db == nil && h.telemetryStore == nil {
			return nil, fmt.Errorf("database unavailable")
		}

		store := h.telemetryStore
		if store == nil {
			store = db.NewADKTelemetryStore(h.db, h.firestoreClient)
		}

		filter := db.TelemetryFilter{
			OrganizationID: reqOrgID,
			ProjectID:      reqProjectID,
			UserID:         reqUserID,
			SessionID:      reqSessionID,
			StartDate:      startDate,
			EndDate:        endDate,
			Limit:          limit,
			Offset:         offset,
		}

		resp, err := store.GetTelemetryUsage(context.WithoutCancel(r.Context()), filter)
		if err != nil {
			log.Printf("[TelemetryHandler] Database query error: %v", err)
			return nil, fmt.Errorf("database unavailable")
		}

		csvBytes, err := db.FormatTelemetryCSV(resp)
		if err != nil {
			return nil, err
		}

		ci := &cachedTelemetryItem{
			response:  resp,
			csvData:   csvBytes,
			createdAt: time.Now(),
		}

		h.cacheMu.Lock()
		h.cache[cacheKey] = ci
		h.cacheMu.Unlock()

		return ci, nil
	})

	if err != nil {
		log.Printf("[TelemetryHandler] Telemetry request error: %v", err)
		sendAPFError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "database unreachable or error: database unavailable")
		return
	}

	ci := res.(*cachedTelemetryItem)
	if format == "csv" {
		fn := "telemetry-usage.csv"
		if reqOrgID != "" {
			fn = fmt.Sprintf("telemetry-%s.csv", reqOrgID)
		} else if reqSessionID != "" {
			fn = fmt.Sprintf("telemetry-%s.csv", reqSessionID)
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fn))
		w.WriteHeader(http.StatusOK)
		w.Write(ci.csvData)
		return
	}

	sendJSON(w, http.StatusOK, ci.response)
}
