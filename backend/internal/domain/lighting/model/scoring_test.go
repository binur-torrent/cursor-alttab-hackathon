package model

import "testing"

func TestScoreSegment(t *testing.T) {
	tests := []struct {
		name          string
		streetlights  int
		lengthM       float64
		roadType      string
		nightRatio    float64
		wantAdequacy  float64
		wantLevel     string
	}{
		{"dark highway is critical", 2, 200, "highway", 0.8, 0.25, "critical"},
		{"well lit residential is low", 5, 200, "residential", 0.2, 1.0, "low"},
		{"unknown road type uses default", 1, 100, "unknown", 0, 0.4, "high"},
		{"no fixtures is critical", 0, 200, "primary", 0, 0.0, "critical"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreSegment(tt.streetlights, tt.lengthM, tt.roadType, tt.nightRatio)
			if got.Adequacy != tt.wantAdequacy {
				t.Errorf("adequacy = %v, want %v", got.Adequacy, tt.wantAdequacy)
			}
			if got.RiskLevel != tt.wantLevel {
				t.Errorf("risk level = %v, want %v (score=%v)", got.RiskLevel, tt.wantLevel, got.RiskScore)
			}
		})
	}
}

func TestSimulateScenarioReducesRisk(t *testing.T) {
	segments := []*StreetSegment{
		{ExternalID: "a", RoadType: "highway", LengthM: 200, StreetLightCount: 1, RiskScore: 100, RiskLevel: "critical"},
		{ExternalID: "b", RoadType: "residential", LengthM: 200, StreetLightCount: 5, RiskScore: 0, RiskLevel: "low"},
	}
	res := SimulateScenario(segments, ScenarioParams{
		TargetRiskLevels:    []string{"critical", "high"},
		TargetAdequacy:      0.9,
		NightDimmingPercent: 50,
	})

	if res.SegmentsUpgraded != 1 {
		t.Errorf("expected 1 segment upgraded, got %d", res.SegmentsUpgraded)
	}
	if res.AddedFixtures <= 0 {
		t.Errorf("expected fixtures to be added, got %d", res.AddedFixtures)
	}
	if res.ProposedAvgRisk >= res.BaselineAvgRisk {
		t.Errorf("expected risk to drop: baseline=%v proposed=%v", res.BaselineAvgRisk, res.ProposedAvgRisk)
	}
	if res.RiskReductionPercent <= 0 {
		t.Errorf("expected positive risk reduction, got %v", res.RiskReductionPercent)
	}
	if res.SegmentsDimmed != 1 {
		t.Errorf("expected 1 low-risk segment dimmed, got %d", res.SegmentsDimmed)
	}
}
