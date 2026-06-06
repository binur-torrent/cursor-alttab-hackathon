package model

import "math"

// Recommendation is a single actionable intervention a municipality can take on
// a segment, with the projected improvement to its overall lighting score.
type Recommendation struct {
	Action                string             `json:"action"` // install_lamps | increase_brightness | trim_vegetation | schedule_inspection
	Title                 string             `json:"title"`
	Detail                string             `json:"detail"`
	Priority              string             `json:"priority"` // high | medium | low
	ProjectedOverallScore float64            `json:"projected_overall_score"`
	ProjectedDelta        float64            `json:"projected_delta"`
	Params                map[string]float64 `json:"params,omitempty"`
}

// targetAdequacy is the design target the install-lamps recommendation aims for.
const targetAdequacy = 0.85

// FeaturesOf builds the scoring input from a stored segment.
func FeaturesOf(s *StreetSegment) Features {
	return Features{
		Streetlights:     s.StreetLightCount,
		PoleCount:        s.PoleCount,
		LengthM:          s.LengthM,
		RoadType:         s.RoadType,
		NightRatio:       s.NightSampleRatio,
		RoadWidthM:       s.RoadWidthM,
		TreeCount:        s.TreeCount,
		VegetationRatio:  s.VegetationRatio,
		BuildingRatio:    s.BuildingRatio,
		SidewalkRatio:    s.SidewalkRatio,
		SkyRatio:         s.SkyRatio,
		BrightnessFactor: s.BrightnessFactor,
	}
}

// ApplyScores writes a computed breakdown back onto a segment.
func ApplyScores(s *StreetSegment, bd RiskBreakdown) {
	s.LightingDensity = bd.Density
	s.RecommendedDensity = bd.Recommended
	s.Adequacy = bd.Adequacy
	s.LightingSufficiency = bd.LightingSufficiency
	s.Occlusion = bd.Occlusion
	s.InfrastructureAdequacy = bd.InfrastructureAdequacy
	s.OverallScore = bd.OverallScore
	s.RiskScore = bd.RiskScore
	s.RiskLevel = bd.RiskLevel
}

// LampsToReachTarget returns how many additional fixtures are needed to bring a
// segment up to the design target adequacy.
func LampsToReachTarget(s *StreetSegment) int {
	recommended := RecommendedDensityFor(s.RoadType)
	needed := int(math.Ceil(targetAdequacy * recommended * (s.LengthM / 100.0)))
	add := needed - s.StreetLightCount
	if add < 0 {
		return 0
	}
	return add
}

// Recommend produces a prioritised set of interventions for a segment, each
// scored with the projected overall lighting score after applying it.
func Recommend(s *StreetSegment) []Recommendation {
	base := FeaturesOf(s)
	current := ScoreEnv(base).OverallScore
	var recs []Recommendation

	// 1. Install additional lamp posts to reach target coverage.
	if add := LampsToReachTarget(s); add > 0 {
		f := base
		f.Streetlights += add
		f.PoleCount += add
		proj := ScoreEnv(f).OverallScore
		recs = append(recs, Recommendation{
			Action:                "install_lamps",
			Title:                 pluralLamps(add),
			Detail:                "Raise streetlight density toward the recommended level for this road class.",
			Priority:              priorityFor(proj - current),
			ProjectedOverallScore: proj,
			ProjectedDelta:        round1(proj - current),
			Params:                map[string]float64{"lamps": float64(add)},
		})
	}

	// 2. Trim vegetation when occlusion is significant.
	if s.Occlusion >= 35 || s.VegetationRatio >= 0.3 {
		f := base
		f.VegetationRatio = clamp(s.VegetationRatio*0.4, 0, 1)
		f.TreeCount = int(math.Round(float64(s.TreeCount) * 0.5))
		proj := ScoreEnv(f).OverallScore
		recs = append(recs, Recommendation{
			Action:                "trim_vegetation",
			Title:                 "Trim vegetation along the segment",
			Detail:                "Tree canopy is blocking lamp output; pruning restores effective illumination.",
			Priority:              priorityFor(proj - current),
			ProjectedOverallScore: proj,
			ProjectedDelta:        round1(proj - current),
			Params:                map[string]float64{"vegetation_factor": 0.4},
		})
	}

	// 3. Increase lamp brightness where lamps exist but coverage is moderate.
	if s.StreetLightCount > 0 && s.Adequacy < 0.9 {
		f := base
		f.BrightnessFactor = 1.3
		proj := ScoreEnv(f).OverallScore
		if proj-current >= 1 {
			recs = append(recs, Recommendation{
				Action:                "increase_brightness",
				Title:                 "Increase lamp brightness (+30%)",
				Detail:                "Uprate existing luminaires before adding new poles - the fastest low-cost gain.",
				Priority:              priorityFor(proj - current),
				ProjectedOverallScore: proj,
				ProjectedDelta:        round1(proj - current),
				Params:                map[string]float64{"brightness_factor": 1.3},
			})
		}
	}

	// 4. Schedule a maintenance / inspection visit when infra is weak.
	if s.InfrastructureAdequacy < 50 || s.StreetLightCount == 0 {
		recs = append(recs, Recommendation{
			Action:                "schedule_inspection",
			Title:                 "Schedule maintenance / field inspection",
			Detail:                "Low infrastructure readiness or a suspected outage - verify poles and fixtures on site.",
			Priority:              "medium",
			ProjectedOverallScore: current,
			ProjectedDelta:        0,
		})
	}

	return recs
}

func pluralLamps(n int) string {
	if n == 1 {
		return "Install 1 additional lamp post"
	}
	return "Install " + itoa(n) + " additional lamp posts"
}

func priorityFor(delta float64) string {
	switch {
	case delta >= 20:
		return "high"
	case delta >= 8:
		return "medium"
	default:
		return "low"
	}
}

// itoa is a tiny dependency-free int->string helper (domain layer stays pure).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
