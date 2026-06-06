package model

// CityStats aggregates network-wide KPIs for the dashboard.
type CityStats struct {
	TotalSegments     int             `json:"total_segments"`
	TotalStreetLights int             `json:"total_street_lights"`
	TotalPoles        int             `json:"total_poles"`
	TotalLengthKm     float64         `json:"total_length_km"`
	AvgRiskScore      float64         `json:"avg_risk_score"`
	ByRiskLevel       map[string]int  `json:"by_risk_level"`
	ByDistrict        []DistrictStat  `json:"by_district"`
	HighRiskSegments  int             `json:"high_risk_segments"`
}

// DistrictStat is per-district rollup used for the dashboard breakdown.
type DistrictStat struct {
	District          string  `json:"district"`
	SegmentCount      int     `json:"segment_count"`
	AvgRiskScore      float64 `json:"avg_risk_score"`
	TotalStreetLights int     `json:"total_street_lights"`
	HighRiskSegments  int     `json:"high_risk_segments"`
}
