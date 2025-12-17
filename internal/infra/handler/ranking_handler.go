package handler

import (
	"net/http"

	usecaseRanking "hateblog/internal/usecase/ranking"
)

const (
	defaultRankingLimit       = 100
	maxYearlyRankingLimit     = 1000
	maxMonthlyRankingLimit    = 100
	maxWeeklyRankingLimit     = 100
	defaultRankingMinBookmark = 5
)

// RankingHandler serves ranking endpoints.
type RankingHandler struct {
	service *usecaseRanking.Service
}

// NewRankingHandler builds a RankingHandler.
func NewRankingHandler(service *usecaseRanking.Service) *RankingHandler {
	return &RankingHandler{service: service}
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
		writeError(w, http.StatusBadRequest, err)
		return
	}
	limit, err := readQueryInt(r, "limit", 1, maxYearlyRankingLimit, defaultRankingLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	minUsers, err := readQueryInt(r, "min_users", 0, 10000, defaultRankingMinBookmark)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.Yearly(r.Context(), year, limit, minUsers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, buildRankingResponse("yearly", year, nil, nil, result))
}

func (h *RankingHandler) handleMonthly(w http.ResponseWriter, r *http.Request) {
	year, err := requireQueryInt(r, "year", 2000, 9999)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	month, err := requireQueryInt(r, "month", 1, 12)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	limit, err := readQueryInt(r, "limit", 1, maxMonthlyRankingLimit, defaultRankingLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	minUsers, err := readQueryInt(r, "min_users", 0, 10000, defaultRankingMinBookmark)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.Monthly(r.Context(), year, month, limit, minUsers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, buildRankingResponse("monthly", year, &month, nil, result))
}

func (h *RankingHandler) handleWeekly(w http.ResponseWriter, r *http.Request) {
	year, err := requireQueryInt(r, "year", 2000, 9999)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	week, err := requireQueryInt(r, "week", 1, 53)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	limit, err := readQueryInt(r, "limit", 1, maxWeeklyRankingLimit, defaultRankingLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	minUsers, err := readQueryInt(r, "min_users", 0, 10000, defaultRankingMinBookmark)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.Weekly(r.Context(), year, week, limit, minUsers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, buildRankingResponse("weekly", year, nil, &week, result))
}

func buildRankingResponse(periodType string, year int, month, week *int, result usecaseRanking.Result) rankingResponse {
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
			Entry: toEntryResponse(ent),
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
