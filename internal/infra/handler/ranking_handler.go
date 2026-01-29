package handler

import (
	"net/http"

	usecaseRanking "hateblog/internal/usecase/ranking"
)

const (
	defaultRankingLimit       = 100
	maxYearlyRankingLimit     = 1000
	maxMonthlyRankingLimit    = 1000
	maxWeeklyRankingLimit     = 1000
	defaultRankingMinBookmark = 5
)

// RankingHandler serves ranking endpoints.
type RankingHandler struct {
	service     *usecaseRanking.Service
	apiBasePath string
}

// NewRankingHandler builds a RankingHandler.
func NewRankingHandler(service *usecaseRanking.Service, apiBasePath string) *RankingHandler {
	return &RankingHandler{
		service:     service,
		apiBasePath: normalizeAPIBasePath(apiBasePath),
	}
}

// RegisterRoutes registers ranking endpoints.
func (h *RankingHandler) RegisterRoutes(r chiRouter) {
	r.Get("/rankings/yearly", h.handleYearly)
	r.Get("/rankings/monthly", h.handleMonthly)
	r.Get("/rankings/weekly", h.handleWeekly)
}

func (h *RankingHandler) handleYearly(w http.ResponseWriter, r *http.Request) {
	year, err := requireQueryInt(r, "year", 2000, 9999)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	limit, err := readQueryInt(r, "limit", 1, maxYearlyRankingLimit, defaultRankingLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	minUsers, err := readQueryInt(r, "min_users", 0, 10000, defaultRankingMinBookmark)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}

	result, cacheHit, err := h.service.YearlyWithCacheStatus(r.Context(), year, limit, minUsers)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	setCacheStatusHeader(w, cacheHit)
	writeJSON(w, http.StatusOK, buildRankingResponse("yearly", year, nil, nil, result, h.apiBasePath))
}

func (h *RankingHandler) handleMonthly(w http.ResponseWriter, r *http.Request) {
	year, err := requireQueryInt(r, "year", 2000, 9999)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	month, err := requireQueryInt(r, "month", 1, 12)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	limit, err := readQueryInt(r, "limit", 1, maxMonthlyRankingLimit, defaultRankingLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	minUsers, err := readQueryInt(r, "min_users", 0, 10000, defaultRankingMinBookmark)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}

	result, cacheHit, err := h.service.MonthlyWithCacheStatus(r.Context(), year, month, limit, minUsers)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	setCacheStatusHeader(w, cacheHit)
	writeJSON(w, http.StatusOK, buildRankingResponse("monthly", year, &month, nil, result, h.apiBasePath))
}

func (h *RankingHandler) handleWeekly(w http.ResponseWriter, r *http.Request) {
	year, err := requireQueryInt(r, "year", 2000, 9999)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	week, err := requireQueryInt(r, "week", 1, 53)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	limit, err := readQueryInt(r, "limit", 1, maxWeeklyRankingLimit, defaultRankingLimit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	minUsers, err := readQueryInt(r, "min_users", 0, 10000, defaultRankingMinBookmark)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}

	result, cacheHit, err := h.service.WeeklyWithCacheStatus(r.Context(), year, week, limit, minUsers)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	setCacheStatusHeader(w, cacheHit)
	writeJSON(w, http.StatusOK, buildRankingResponse("weekly", year, nil, &week, result, h.apiBasePath))
}

func buildRankingResponse(periodType string, year int, month, week *int, result usecaseRanking.Result, apiBasePath string) rankingResponse {
	resp := rankingResponse{
		PeriodType: periodType,
		Year:       year,
		Entries:    make([]rankingEntryResponse, 0, len(result.Entries)),
		Total:      result.Total,
	}
	if month != nil {
		resp.Month = month
	}
	if week != nil {
		resp.Week = week
	}
	for i, ent := range result.Entries {
		resp.Entries = append(resp.Entries, rankingEntryResponse{
			Rank:  i + 1,
			Entry: toEntryResponse(ent, apiBasePath),
		})
	}
	return resp
}

type rankingResponse struct {
	PeriodType string                 `json:"period_type"`
	Year       int                    `json:"year"`
	Month      *int                   `json:"month,omitempty"`
	Week       *int                   `json:"week,omitempty"`
	Entries    []rankingEntryResponse `json:"entries"`
	Total      int64                  `json:"total"`
}

type rankingEntryResponse struct {
	Rank  int           `json:"rank"`
	Entry entryResponse `json:"entry"`
}
