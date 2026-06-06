// Package lighting provides HTTP handlers for the lighting bounded context.
package lighting

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/masterfabric-go/masterfabric/internal/application/lighting/dto"
	"github.com/masterfabric-go/masterfabric/internal/application/lighting/usecase"
	"github.com/masterfabric-go/masterfabric/internal/domain/lighting/repository"
	"github.com/masterfabric-go/masterfabric/internal/shared/pagination"
	"github.com/masterfabric-go/masterfabric/internal/shared/response"
	"github.com/masterfabric-go/masterfabric/internal/shared/validator"
)

// Handler provides Lighting HTTP handlers.
type Handler struct {
	listSegmentsUC *usecase.ListSegmentsUseCase
	getSegmentUC   *usecase.GetSegmentUseCase
	getStatsUC     *usecase.GetStatsUseCase
	simulateUC     *usecase.SimulateScenarioUseCase
	analyzeLiveUC  *usecase.AnalyzeLiveUseCase
	rescoreUC      *usecase.RescoreSegmentUseCase
}

// NewHandler creates a new Lighting handler.
func NewHandler(
	listSegmentsUC *usecase.ListSegmentsUseCase,
	getSegmentUC *usecase.GetSegmentUseCase,
	getStatsUC *usecase.GetStatsUseCase,
	simulateUC *usecase.SimulateScenarioUseCase,
	analyzeLiveUC *usecase.AnalyzeLiveUseCase,
	rescoreUC *usecase.RescoreSegmentUseCase,
) *Handler {
	return &Handler{
		listSegmentsUC: listSegmentsUC,
		getSegmentUC:   getSegmentUC,
		getStatsUC:     getStatsUC,
		simulateUC:     simulateUC,
		analyzeLiveUC:  analyzeLiveUC,
		rescoreUC:      rescoreUC,
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

// Simulate runs an adaptive-lighting policy simulation (energy vs safety).
func (h *Handler) Simulate(w http.ResponseWriter, r *http.Request) {
	var req dto.SimulateRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.simulateUC.Execute(
		r.Context(),
		req.ScenarioParams,
		repository.SegmentFilter{District: req.District},
	)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// Analyze runs on-demand CV analysis of a single point (Street View via the AI
// worker, or a deterministic fallback). Faces/plates are anonymized upstream.
func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	var req dto.AnalyzeRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.analyzeLiveUC.Execute(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// Rescore applies a what-if intervention to a segment (add lamps, trim
// vegetation, change brightness) and returns the baseline vs projected scores
// plus recommendations. With persist=true it writes the new state back so the
// map recolors. Accepts a UUID or external id in the path.
func (h *Handler) Rescore(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	var req dto.RescoreRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.rescoreUC.Execute(r.Context(), idParam, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
