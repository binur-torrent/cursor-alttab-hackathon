package dto

import "github.com/masterfabric-go/masterfabric/internal/domain/lighting/model"

// SimulateRequest is the body for POST /lighting/simulate. It embeds the domain
// scenario parameters and adds an optional district scope.
type SimulateRequest struct {
	model.ScenarioParams
	District string `json:"district"`
}
