package model

import "testing"

func TestScoreSegmentAdequacy(t *testing.T) {
	tests := []struct {
		name         string
		streetlights int
		lengthM      float64
		roadType     string
		nightRatio   float64
		wantAdequacy float64
	}{
		{"dark highway underlit", 2, 200, "highway", 0.8, 0.25},
		{"well lit residential", 5, 200, "residential", 0.2, 1.0},
		{"unknown road type uses default", 1, 100, "unknown", 0, 0.4},
		{"no fixtures", 0, 200, "primary", 0, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreSegment(tt.streetlights, tt.lengthM, tt.roadType, tt.nightRatio)
			if got.Adequacy != tt.wantAdequacy {
				t.Errorf("adequacy = %v, want %v", got.Adequacy, tt.wantAdequacy)
			}
		})
	}
}

// TestScoreEnvOcclusionLowersScore verifies that heavy vegetation occlusion
// pushes a segment toward higher risk even when lamp count is unchanged.
func TestScoreEnvOcclusionLowersScore(t *testing.T) {
	clear := ScoreEnv(Features{Streetlights: 4, LengthM: 200, RoadType: "secondary", PoleCount: 4})
	occluded := ScoreEnv(Features{
		Streetlights: 4, LengthM: 200, RoadType: "secondary", PoleCount: 4,
		VegetationRatio: 0.85, TreeCount: 8, BuildingRatio: 0.4,
	})
	if occluded.Occlusion <= clear.Occlusion {
		t.Errorf("expected higher occlusion, got %v vs %v", occluded.Occlusion, clear.Occlusion)
	}
	if occluded.OverallScore >= clear.OverallScore {
		t.Errorf("occlusion should lower overall score: %v vs %v", occluded.OverallScore, clear.OverallScore)
	}
}

// TestScoreEnvAddingLampsImprovesScore is the core "adjust the rate" guarantee:
// installing lamps must raise the overall score.
func TestScoreEnvAddingLampsImprovesScore(t *testing.T) {
	before := ScoreEnv(Features{Streetlights: 1, LengthM: 200, RoadType: "primary", NightRatio: 0.8})
	after := ScoreEnv(Features{Streetlights: 6, PoleCount: 6, LengthM: 200, RoadType: "primary", NightRatio: 0.8})
	if after.OverallScore <= before.OverallScore {
		t.Errorf("adding lamps should improve overall: before=%v after=%v", before.OverallScore, after.OverallScore)
	}
	if after.LightingSufficiency <= before.LightingSufficiency {
		t.Errorf("adding lamps should improve sufficiency: before=%v after=%v",
			before.LightingSufficiency, after.LightingSufficiency)
	}
}

func TestRecommendInstallsLampsWhenUnderlit(t *testing.T) {
	s := &StreetSegment{
		ExternalID: "x", RoadType: "primary", LengthM: 200, StreetLightCount: 1,
		PoleCount: 1, NightSampleRatio: 0.8, VegetationRatio: 0.6, TreeCount: 6,
	}
	ApplyScores(s, ScoreEnv(FeaturesOf(s)))
	recs := Recommend(s)
	if len(recs) == 0 {
		t.Fatal("expected recommendations for an underlit, occluded segment")
	}
	var hasLamps, hasTrim bool
	for _, r := range recs {
		if r.Action == "install_lamps" {
			hasLamps = true
			if r.ProjectedDelta <= 0 {
				t.Errorf("install_lamps should project a positive delta, got %v", r.ProjectedDelta)
			}
		}
		if r.Action == "trim_vegetation" {
			hasTrim = true
		}
	}
	if !hasLamps {
		t.Error("expected an install_lamps recommendation")
	}
	if !hasTrim {
		t.Error("expected a trim_vegetation recommendation")
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
