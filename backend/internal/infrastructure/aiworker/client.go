// Package aiworker is the HTTP adapter to the LumiCity Python AI worker
// (FastAPI). It implements usecase.AnalyzerPort.
package aiworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/masterfabric-go/masterfabric/internal/application/lighting/dto"
)

// Client calls the AI worker's /analyze/point endpoint.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a worker client for the given base URL (e.g.
// https://lumicity-ai-worker.onrender.com).
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

type pointRequest struct {
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Heading  float64 `json:"heading"`
	RoadType string  `json:"road_type"`
	LengthM  float64 `json:"length_m"`
	IsNight  bool    `json:"is_night"`
}

// AnalyzePoint sends the point to the worker and maps the response.
func (c *Client) AnalyzePoint(ctx context.Context, req dto.AnalyzeRequest) (*dto.AnalyzeResult, error) {
	body, err := json.Marshal(pointRequest{
		Lat:      req.Lat,
		Lon:      req.Lon,
		Heading:  req.Heading,
		RoadType: req.RoadType,
		LengthM:  req.LengthM,
		IsNight:  req.IsNight,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/analyze/point", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai worker returned status %d", resp.StatusCode)
	}

	var result dto.AnalyzeResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
