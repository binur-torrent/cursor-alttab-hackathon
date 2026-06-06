// Package lighting provides HTTP handlers for the lighting bounded context.
package lighting

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/masterfabric-go/masterfabric/internal/application/lighting/usecase"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/repository"
	"github.com/masterfabric-go/masterfabric/internal/shared/pagination"
	"github.com/masterfabric-go/masterfabric/internal/shared/response"
)

// Handler provides Lighting HTTP handlers.
type Handler struct {
	listSegmentsUC *usecase.ListSegmentsUseCase
	getSegmentUC   *usecase.GetSegmentUseCase
	getStatsUC     *usecase.GetStatsUseCase
}

// NewHandler creates a new Lighting handler.
func NewHandler(
	listSegmentsUC *usecase.ListSegmentsUseCase,
	getSegmentUC *usecase.GetSegmentUseCase,
	getStatsUC *usecase.GetStatsUseCase,
) *Handler {
	return &Handler{
		listSegmentsUC: listSegmentsUC,
		getSegmentUC:   getSegmentUC,
		getStatsUC:     getStatsUC,
	}
}

func filterFromRequest(r *http.Request) repository.SegmentFilter {
	q := r.URL.Query()
	minRisk, _ := strconv.ParseFloat(q.Get("min_risk"), 64)
	return repository.SegmentFilter{
		District:  q.Get("district"),
		RoadType:  q.Get("road_type"),
		RiskLevel: q.Get("risk_level"),
		MinRisk:   minRisk,
	}
}

// ListSegments returns a paginated, filtered list of segments (highest risk first).
func (h *Handler) ListSegments(w http.ResponseWriter, r *http.Request) {
	params := pagination.FromRequest(r)
	filter := filterFromRequest(r)

	segments, total, err := h.listSegmentsUC.Execute(r.Context(), filter, params.Offset(), params.Limit())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, pagination.NewResult(segments, params, total))
}

// ListMap returns every matching segment (with geometry) for the map view.
func (h *Handler) ListMap(w http.ResponseWriter, r *http.Request) {
	filter := filterFromRequest(r)
	segments, err := h.listSegmentsUC.ExecuteAll(r.Context(), filter)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"count":    len(segments),
		"segments": segments,
	})
}

// GetSegment returns one segment's full detail. Accepts either a UUID or an
// external id (e.g. "seg-00-01").
func (h *Handler) GetSegment(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	if id, err := uuid.Parse(idParam); err == nil {
		detail, err := h.getSegmentUC.Execute(r.Context(), id)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.JSON(w, http.StatusOK, detail)
		return
	}

	detail, err := h.getSegmentUC.ExecuteByExternalID(r.Context(), idParam)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, detail)
}

// GetStats returns network-wide lighting KPIs.
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.getStatsUC.Execute(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, stats)
}
