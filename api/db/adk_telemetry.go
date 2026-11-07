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

package db

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"gorm.io/gorm"
)

// ADKPostgresSession represents the adk_sessions GORM model.
type ADKPostgresSession struct {
	SessionID      string    `gorm:"column:session_id;primaryKey" json:"sessionId"`
	UserID         string    `gorm:"column:user_id;index" json:"userId"`
	ProjectID      string    `gorm:"column:project_id;index" json:"projectId"`
	OrganizationID string    `gorm:"column:organization_id;index" json:"organizationId"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

func (ADKPostgresSession) TableName() string {
	return "adk_sessions"
}

type ADKSession = ADKPostgresSession

// ADKTelemetryEvent represents the adk_events GORM model.
type ADKTelemetryEvent struct {
	ID               string    `gorm:"column:id;primaryKey" json:"id"`
	SessionID        string    `gorm:"column:session_id;index" json:"sessionId"`
	UserID           string    `gorm:"column:user_id;index" json:"userId"`
	ProjectID        string    `gorm:"column:project_id;index" json:"projectId"`
	OrganizationID   string    `gorm:"column:organization_id;index" json:"organizationId"`
	Model            string    `gorm:"column:model" json:"model"`
	PromptTokens     int64     `gorm:"column:prompt_tokens" json:"promptTokens"`
	CompletionTokens int64     `gorm:"column:completion_tokens" json:"completionTokens"`
	TotalTokens      int64     `gorm:"column:total_tokens" json:"totalTokens"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"createdAt"`
	IsTransient      bool      `gorm:"column:is_transient" json:"isTransient"`
}

func (ADKTelemetryEvent) TableName() string {
	return "adk_events"
}

type ADKPostgresEvent = ADKTelemetryEvent

// ModelRate defines pricing per 1K tokens in USD.
type ModelRate struct {
	PromptTokenRateUSD     float64 `firestore:"promptTokenRateUSD" json:"promptTokenRateUSD"`
	CompletionTokenRateUSD float64 `firestore:"completionTokenRateUSD" json:"completionTokenRateUSD"`
}

// Default baseline rates ($ per 1K tokens)
var DefaultBaselineRate = ModelRate{
	PromptTokenRateUSD:     0.000125, // $0.125 / 1M tokens
	CompletionTokenRateUSD: 0.000375, // $0.375 / 1M tokens
}

// Known model rate cards ($ per 1K tokens)
var KnownBaselineRateCards = map[string]ModelRate{
	"gemini-1.5-pro": {
		PromptTokenRateUSD:     0.00125,
		CompletionTokenRateUSD: 0.00500,
	},
	"gemini-1.5-flash": {
		PromptTokenRateUSD:     0.000075,
		CompletionTokenRateUSD: 0.000300,
	},
	"gemini-2.0-flash": {
		PromptTokenRateUSD:     0.000100,
		CompletionTokenRateUSD: 0.000400,
	},
	"gemini-3.7-flash": {
		PromptTokenRateUSD:     0.000100,
		CompletionTokenRateUSD: 0.000400,
	},
}

// GetModelRate fetches rate card from Firestore configs/global or falls back to baseline rates.
func GetModelRate(ctx context.Context, firestoreClient *firestore.Client, modelID string) ModelRate {
	if modelID == "" {
		modelID = "default"
	}

	if firestoreClient != nil {
		doc, err := firestoreClient.Collection("configs").Doc("global").Get(ctx)
		if err == nil && doc.Exists() {
			data := doc.Data()
			if rateCardsRaw, ok := data["rateCards"].(map[string]interface{}); ok {
				if modelDataRaw, ok := rateCardsRaw[modelID].(map[string]interface{}); ok {
					var rate ModelRate
					if pRate, ok := modelDataRaw["promptTokenRateUSD"].(float64); ok {
						rate.PromptTokenRateUSD = pRate
					}
					if cRate, ok := modelDataRaw["completionTokenRateUSD"].(float64); ok {
						rate.CompletionTokenRateUSD = cRate
					}
					if rate.PromptTokenRateUSD > 0 || rate.CompletionTokenRateUSD > 0 {
						return rate
					}
				}
			}
		}
	}

	// Check known baseline rate cards
	if known, ok := KnownBaselineRateCards[modelID]; ok {
		return known
	}

	// Unknown model fallback: log warning and return default baseline rate
	log.Printf("WARNING: Unknown model ID %q, falling back to default baseline rate card", modelID)
	return DefaultBaselineRate
}

// RoundUSD rounds a float64 value to 4 decimal places.
func RoundUSD(val float64) float64 {
	return math.Round(val*10000.0) / 10000.0
}

// TelemetryFilter specifies query filters for telemetry aggregations.
type TelemetryFilter struct {
	OrganizationID string
	ProjectID      string
	UserID         string
	SessionID      string
	StartDate      time.Time
	EndDate        time.Time
	Limit          int
	Offset         int
}

// TelemetryUsageResponse holds aggregated usage metrics and breakdowns.
type TelemetryUsageResponse struct {
	OrganizationID       string                      `json:"organizationId,omitempty"`
	ProjectID            string                      `json:"projectId,omitempty"`
	UserID               string                      `json:"userId,omitempty"`
	SessionID            string                      `json:"sessionId,omitempty"`
	StartDate            string                      `json:"startDate"`
	EndDate              string                      `json:"endDate"`
	PromptTokens         int64                       `json:"promptTokens"`
	CompletionTokens     int64                       `json:"completionTokens"`
	TotalTokens          int64                       `json:"totalTokens"`
	CostUSD              float64                     `json:"costUSD"`
	TurnCount            int64                       `json:"turnCount,omitempty"`
	AverageTokensPerTurn float64                     `json:"averageTokensPerTurn,omitempty"`
	UserBreakdown        []UserTelemetryBreakdown    `json:"userBreakdown,omitempty"`
	ProjectBreakdown     []ProjectTelemetryBreakdown `json:"projectBreakdown,omitempty"`
	SessionBreakdown     []SessionTelemetryBreakdown `json:"sessionBreakdown,omitempty"`
	TotalUsers           int64                       `json:"totalUsers,omitempty"`
	TotalProjects        int64                       `json:"totalProjects,omitempty"`
	Limit                int                         `json:"limit,omitempty"`
	Offset               int                         `json:"offset,omitempty"`
}

type UserTelemetryBreakdown struct {
	UserID           string  `json:"userId"`
	PromptTokens     int64   `json:"promptTokens"`
	CompletionTokens int64   `json:"completionTokens"`
	TotalTokens      int64   `json:"totalTokens"`
	CostUSD          float64 `json:"costUSD"`
	SessionCount     int64   `json:"sessionCount"`
}

type ProjectTelemetryBreakdown struct {
	ProjectID        string  `json:"projectId"`
	PromptTokens     int64   `json:"promptTokens"`
	CompletionTokens int64   `json:"completionTokens"`
	TotalTokens      int64   `json:"totalTokens"`
	CostUSD          float64 `json:"costUSD"`
	SessionCount     int64   `json:"sessionCount"`
}

type SessionTelemetryBreakdown struct {
	SessionID        string  `json:"sessionId"`
	UserID           string  `json:"userId"`
	ProjectID        string  `json:"projectId"`
	PromptTokens     int64   `json:"promptTokens"`
	CompletionTokens int64   `json:"completionTokens"`
	TotalTokens      int64   `json:"totalTokens"`
	CostUSD          float64 `json:"costUSD"`
	TurnCount        int64   `json:"turnCount"`
}

// ADKTelemetryStore handles telemetry database queries and aggregations.
type ADKTelemetryStore struct {
	db              *gorm.DB
	firestoreClient *firestore.Client
}

// NewADKTelemetryStore constructs a new ADKTelemetryStore instance.
func NewADKTelemetryStore(db *gorm.DB, firestoreClient *firestore.Client) *ADKTelemetryStore {
	return &ADKTelemetryStore{
		db:              db,
		firestoreClient: firestoreClient,
	}
}

type EventModelGroup struct {
	Model            string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	TurnCount        int64
}

type UserGroupResult struct {
	UserID           string
	Model            string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	SessionCount     int64
}

type ProjectGroupResult struct {
	ProjectID        string
	Model            string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	SessionCount     int64
}

type SessionGroupResult struct {
	SessionID        string
	UserID           string
	ProjectID        string
	Model            string
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	TurnCount        int64
}

// GetTelemetryUsage queries the ADK Session DB and returns aggregated telemetry.
func (s *ADKTelemetryStore) GetTelemetryUsage(ctx context.Context, filter TelemetryFilter) (*TelemetryUsageResponse, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	// Cap pagination limit at 500
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		log.Printf("WARNING: Telemetry limit %d exceeds max 500, capping to 500", limit)
		limit = 500
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Default trailing 30-day window
	endDate := filter.EndDate
	if endDate.IsZero() {
		endDate = time.Now()
	}
	startDate := filter.StartDate
	if startDate.IsZero() {
		startDate = endDate.AddDate(0, 0, -30)
	}

	resp := &TelemetryUsageResponse{
		OrganizationID: filter.OrganizationID,
		ProjectID:      filter.ProjectID,
		UserID:         filter.UserID,
		SessionID:      filter.SessionID,
		StartDate:      startDate.Format(time.RFC3339),
		EndDate:        endDate.Format(time.RFC3339),
		Limit:          limit,
		Offset:         offset,
	}

	// Query directly from existing ADK database tables 'events' and 'sessions'
	eventsTable := "events"
	timeCol := "events.timestamp"
	userCol := "COALESCE(NULLIF(events.user_id, ''), NULLIF(sessions.user_id, ''), 'unknown')"
	idCol := "events.id"
	sessIdCol := "events.session_id"
	joinClause := "LEFT JOIN sessions ON events.session_id = sessions.id"

	projCol := "COALESCE(NULLIF(sessions.state->>'projectId', ''), 'unknown')"
	orgCol := "COALESCE(NULLIF(sessions.state->>'organizationId', ''), '')"
	promptCol := "COALESCE((events.usage_metadata->>'promptTokenCount')::bigint, 0)"
	completionCol := "COALESCE((events.usage_metadata->>'candidatesTokenCount')::bigint, 0)"
	totalCol := "COALESCE((events.usage_metadata->>'totalTokenCount')::bigint, 0)"
	modelCol := "COALESCE(NULLIF(events.custom_metadata->>'model', ''), 'gemini-2.5-pro')"

	if s.db != nil && s.db.Dialector != nil && s.db.Dialector.Name() == "sqlite" {
		projCol = "COALESCE(NULLIF(json_extract(sessions.state, '$.projectId'), ''), 'unknown')"
		orgCol = "COALESCE(NULLIF(json_extract(sessions.state, '$.organizationId'), ''), '')"
		promptCol = "COALESCE(CAST(json_extract(events.usage_metadata, '$.promptTokenCount') AS INTEGER), 0)"
		completionCol = "COALESCE(CAST(json_extract(events.usage_metadata, '$.candidatesTokenCount') AS INTEGER), 0)"
		totalCol = "COALESCE(CAST(json_extract(events.usage_metadata, '$.totalTokenCount') AS INTEGER), 0)"
		modelCol = "COALESCE(NULLIF(json_extract(events.custom_metadata, '$.model'), ''), 'gemini-2.5-pro')"
	}

	// Build base query
	query := s.db.WithContext(ctx).Table(eventsTable).
		Joins(joinClause).
		Where(fmt.Sprintf("%s >= ? AND %s <= ?", timeCol, timeCol), startDate, endDate).
		Where("events.usage_metadata IS NOT NULL")

	if filter.OrganizationID != "" {
		query = query.Where(fmt.Sprintf("%s = ?", orgCol), filter.OrganizationID)
	}
	if filter.ProjectID != "" {
		query = query.Where(fmt.Sprintf("%s = ?", projCol), filter.ProjectID)
	}
	if filter.UserID != "" {
		query = query.Where(fmt.Sprintf("%s = ?", userCol), filter.UserID)
	}
	if filter.SessionID != "" {
		query = query.Where(fmt.Sprintf("%s = ?", sessIdCol), filter.SessionID)
	}

	// General Rollup Query grouped by model
	var modelGroups []EventModelGroup
	selectExpr := fmt.Sprintf("%s as model, SUM(%s) as prompt_tokens, SUM(%s) as completion_tokens, SUM(%s) as total_tokens, COUNT(%s) as turn_count",
		modelCol, promptCol, completionCol, totalCol, idCol)

	err := query.Select(selectExpr).Group(modelCol).Scan(&modelGroups).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query telemetry aggregate: %w", err)
	}

	var totalPrompt, totalCompletion, totalTokens, totalTurns int64
	var rawCost float64

	for _, mg := range modelGroups {
		totalPrompt += mg.PromptTokens
		totalCompletion += mg.CompletionTokens
		totalTokens += mg.TotalTokens
		totalTurns += mg.TurnCount

		rate := GetModelRate(ctx, s.firestoreClient, mg.Model)
		cost := (float64(mg.PromptTokens)/1000.0)*rate.PromptTokenRateUSD + (float64(mg.CompletionTokens)/1000.0)*rate.CompletionTokenRateUSD
		rawCost += cost
	}

	resp.PromptTokens = totalPrompt
	resp.CompletionTokens = totalCompletion
	resp.TotalTokens = totalTokens
	resp.CostUSD = RoundUSD(rawCost)
	resp.TurnCount = totalTurns

	// User Breakdown
	var userResults []UserGroupResult
	uQuery := s.db.WithContext(ctx).Table(eventsTable).
		Joins(joinClause).
		Where(fmt.Sprintf("%s >= ? AND %s <= ?", timeCol, timeCol), startDate, endDate).
		Where("events.usage_metadata IS NOT NULL")

	if filter.OrganizationID != "" {
		uQuery = uQuery.Where(fmt.Sprintf("%s = ?", orgCol), filter.OrganizationID)
	}
	if filter.ProjectID != "" {
		uQuery = uQuery.Where(fmt.Sprintf("%s = ?", projCol), filter.ProjectID)
	}
	if filter.UserID != "" {
		uQuery = uQuery.Where(fmt.Sprintf("%s = ?", userCol), filter.UserID)
	}

	uSelectExpr := fmt.Sprintf("%s as user_id, %s as model, SUM(%s) as prompt_tokens, SUM(%s) as completion_tokens, SUM(%s) as total_tokens, COUNT(DISTINCT %s) as session_count",
		userCol, modelCol, promptCol, completionCol, totalCol, sessIdCol)

	err = uQuery.Select(uSelectExpr).Group(fmt.Sprintf("%s, %s", userCol, modelCol)).Scan(&userResults).Error

	if err == nil {
		userMap := make(map[string]*UserTelemetryBreakdown)
		var userOrder []string

		for _, ur := range userResults {
			if ur.UserID == "" {
				ur.UserID = "unknown"
			}
			ub, exists := userMap[ur.UserID]
			if !exists {
				ub = &UserTelemetryBreakdown{
					UserID:       ur.UserID,
					SessionCount: ur.SessionCount,
				}
				userMap[ur.UserID] = ub
				userOrder = append(userOrder, ur.UserID)
			}
			ub.PromptTokens += ur.PromptTokens
			ub.CompletionTokens += ur.CompletionTokens
			ub.TotalTokens += ur.TotalTokens
			if ur.SessionCount > ub.SessionCount {
				ub.SessionCount = ur.SessionCount
			}

			rate := GetModelRate(ctx, s.firestoreClient, ur.Model)
			cost := (float64(ur.PromptTokens)/1000.0)*rate.PromptTokenRateUSD + (float64(ur.CompletionTokens)/1000.0)*rate.CompletionTokenRateUSD
			ub.CostUSD += cost
		}

		resp.TotalUsers = int64(len(userOrder))
		var userBreakdowns []UserTelemetryBreakdown
		for i, uid := range userOrder {
			if i < offset {
				continue
			}
			if len(userBreakdowns) >= limit {
				break
			}
			ub := userMap[uid]
			ub.CostUSD = RoundUSD(ub.CostUSD)
			userBreakdowns = append(userBreakdowns, *ub)
		}
		resp.UserBreakdown = userBreakdowns
	}

	// Project Breakdown
	var projResults []ProjectGroupResult
	pQuery := s.db.WithContext(ctx).Table(eventsTable).
		Joins(joinClause).
		Where(fmt.Sprintf("%s >= ? AND %s <= ?", timeCol, timeCol), startDate, endDate).
		Where("events.usage_metadata IS NOT NULL")

	if filter.OrganizationID != "" {
		pQuery = pQuery.Where(fmt.Sprintf("%s = ?", orgCol), filter.OrganizationID)
	}
	if filter.ProjectID != "" {
		pQuery = pQuery.Where(fmt.Sprintf("%s = ?", projCol), filter.ProjectID)
	}
	if filter.UserID != "" {
		pQuery = pQuery.Where(fmt.Sprintf("%s = ?", userCol), filter.UserID)
	}

	pSelectExpr := fmt.Sprintf("%s as project_id, %s as model, SUM(%s) as prompt_tokens, SUM(%s) as completion_tokens, SUM(%s) as total_tokens, COUNT(DISTINCT %s) as session_count",
		projCol, modelCol, promptCol, completionCol, totalCol, sessIdCol)

	err = pQuery.Select(pSelectExpr).Group(fmt.Sprintf("%s, %s", projCol, modelCol)).Scan(&projResults).Error

	if err == nil {
		projMap := make(map[string]*ProjectTelemetryBreakdown)
		var projOrder []string

		for _, pr := range projResults {
			if pr.ProjectID == "" {
				pr.ProjectID = "unknown"
			}
			pb, exists := projMap[pr.ProjectID]
			if !exists {
				pb = &ProjectTelemetryBreakdown{
					ProjectID:    pr.ProjectID,
					SessionCount: pr.SessionCount,
				}
				projMap[pr.ProjectID] = pb
				projOrder = append(projOrder, pr.ProjectID)
			}
			pb.PromptTokens += pr.PromptTokens
			pb.CompletionTokens += pr.CompletionTokens
			pb.TotalTokens += pr.TotalTokens
			if pr.SessionCount > pb.SessionCount {
				pb.SessionCount = pr.SessionCount
			}

			rate := GetModelRate(ctx, s.firestoreClient, pr.Model)
			cost := (float64(pr.PromptTokens)/1000.0)*rate.PromptTokenRateUSD + (float64(pr.CompletionTokens)/1000.0)*rate.CompletionTokenRateUSD
			pb.CostUSD += cost
		}

		resp.TotalProjects = int64(len(projOrder))
		var projBreakdowns []ProjectTelemetryBreakdown
		for i, pid := range projOrder {
			if i < offset {
				continue
			}
			if len(projBreakdowns) >= limit {
				break
			}
			pb := projMap[pid]
			pb.CostUSD = RoundUSD(pb.CostUSD)
			projBreakdowns = append(projBreakdowns, *pb)
		}
		resp.ProjectBreakdown = projBreakdowns
	}

	// Session Breakdown
	var sessResults []SessionGroupResult
	sQuery := s.db.WithContext(ctx).Table(eventsTable).
		Joins(joinClause).
		Where(fmt.Sprintf("%s >= ? AND %s <= ?", timeCol, timeCol), startDate, endDate).
		Where("events.usage_metadata IS NOT NULL")

	if filter.OrganizationID != "" {
		sQuery = sQuery.Where(fmt.Sprintf("%s = ?", orgCol), filter.OrganizationID)
	}
	if filter.ProjectID != "" {
		sQuery = sQuery.Where(fmt.Sprintf("%s = ?", projCol), filter.ProjectID)
	}
	if filter.UserID != "" {
		sQuery = sQuery.Where(fmt.Sprintf("%s = ?", userCol), filter.UserID)
	}

	sSelectExpr := fmt.Sprintf("%s as session_id, %s as user_id, %s as project_id, %s as model, SUM(%s) as prompt_tokens, SUM(%s) as completion_tokens, SUM(%s) as total_tokens, COUNT(%s) as turn_count",
		sessIdCol, userCol, projCol, modelCol, promptCol, completionCol, totalCol, idCol)

	err = sQuery.Select(sSelectExpr).Group(fmt.Sprintf("%s, %s, %s, %s", sessIdCol, userCol, projCol, modelCol)).Scan(&sessResults).Error

	if err == nil {
		sessMap := make(map[string]*SessionTelemetryBreakdown)
		var sessOrder []string

		for _, sr := range sessResults {
			if sr.SessionID == "" {
				continue
			}
			sb, exists := sessMap[sr.SessionID]
			if !exists {
				sb = &SessionTelemetryBreakdown{
					SessionID: sr.SessionID,
					UserID:    sr.UserID,
					ProjectID: sr.ProjectID,
				}
				sessMap[sr.SessionID] = sb
				sessOrder = append(sessOrder, sr.SessionID)
			}
			sb.PromptTokens += sr.PromptTokens
			sb.CompletionTokens += sr.CompletionTokens
			sb.TotalTokens += sr.TotalTokens
			sb.TurnCount += sr.TurnCount

			rate := GetModelRate(ctx, s.firestoreClient, sr.Model)
			cost := (float64(sr.PromptTokens)/1000.0)*rate.PromptTokenRateUSD + (float64(sr.CompletionTokens)/1000.0)*rate.CompletionTokenRateUSD
			sb.CostUSD += cost
		}

		var sessBreakdowns []SessionTelemetryBreakdown
		for i, sid := range sessOrder {
			if i < offset {
				continue
			}
			if len(sessBreakdowns) >= limit {
				break
			}
			sb := sessMap[sid]
			sb.CostUSD = RoundUSD(sb.CostUSD)
			sessBreakdowns = append(sessBreakdowns, *sb)
		}
		resp.SessionBreakdown = sessBreakdowns
	}

	if resp.TurnCount > 0 {
		resp.AverageTokensPerTurn = RoundUSD(float64(resp.TotalTokens) / float64(resp.TurnCount))
	}

	return resp, nil
}

func sanitizeCSVField(s string) string {
	if len(s) > 0 && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
		return "'" + s
	}
	return s
}

// FormatTelemetryCSV renders telemetry metrics and breakdowns as a CSV document.
func FormatTelemetryCSV(resp *TelemetryUsageResponse) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("telemetry usage response is nil")
	}

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Write Summary Header and Data
	writer.Write([]string{"Type", "ID", "Input Tokens", "Output Tokens", "Total Tokens", "Cost (USD)", "Sessions/Turns"})
	targetID := sanitizeCSVField(resp.OrganizationID)
	if targetID == "" {
		targetID = sanitizeCSVField(resp.ProjectID)
	}
	if targetID == "" {
		targetID = sanitizeCSVField(resp.SessionID)
	}
	if targetID == "" {
		targetID = "all"
	}

	turnOrSess := resp.TurnCount

	writer.Write([]string{
		"Summary",
		targetID,
		strconv.FormatInt(resp.PromptTokens, 10),
		strconv.FormatInt(resp.CompletionTokens, 10),
		strconv.FormatInt(resp.TotalTokens, 10),
		strconv.FormatFloat(resp.CostUSD, 'f', 4, 64),
		strconv.FormatInt(turnOrSess, 10),
	})

	// User Breakdown
	for _, u := range resp.UserBreakdown {
		writer.Write([]string{
			"User",
			sanitizeCSVField(u.UserID),
			strconv.FormatInt(u.PromptTokens, 10),
			strconv.FormatInt(u.CompletionTokens, 10),
			strconv.FormatInt(u.TotalTokens, 10),
			strconv.FormatFloat(u.CostUSD, 'f', 4, 64),
			strconv.FormatInt(u.SessionCount, 10),
		})
	}

	// Project Breakdown
	for _, p := range resp.ProjectBreakdown {
		writer.Write([]string{
			"Project",
			sanitizeCSVField(p.ProjectID),
			strconv.FormatInt(p.PromptTokens, 10),
			strconv.FormatInt(p.CompletionTokens, 10),
			strconv.FormatInt(p.TotalTokens, 10),
			strconv.FormatFloat(p.CostUSD, 'f', 4, 64),
			strconv.FormatInt(p.SessionCount, 10),
		})
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to format csv: %w", err)
	}

	return buf.Bytes(), nil
}
