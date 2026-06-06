package model

import "math"

// ScenarioParams describes an adaptive-lighting policy to simulate over the
// network. Two levers: (1) upgrade underlit, high-risk segments toward a target
// adequacy, and (2) dim already well-lit segments late at night to save energy.
type ScenarioParams struct {
	// Segments at these risk levels are upgraded toward TargetAdequacy.
	TargetRiskLevels []string `json:"target_risk_levels"`
	// Bring targeted segments up to this lighting adequacy (0..1).
	TargetAdequacy float64 `json:"target_adequacy"`
	// Late-night dimming applied to well-lit, non-targeted segments (0..100 %).
	NightDimmingPercent float64 `json:"night_dimming_percent"`

	// Energy model parameters (sensible LED defaults).
	FixtureWattage   float64 `json:"fixture_wattage"`     // watts per fixture
	HoursPerNight    float64 `json:"hours_per_night"`     // lit hours/night
	EnergyCostPerKwh float64 `json:"energy_cost_per_kwh"` // currency/kWh
	CO2KgPerKwh      float64 `json:"co2_kg_per_kwh"`      // grid emission factor
}

// Normalize fills in defaults for any unset parameters.
func (p *ScenarioParams) Normalize() {
	if len(p.TargetRiskLevels) == 0 {
		p.TargetRiskLevels = []string{string(RiskCritical), string(RiskHigh)}
	}
	if p.TargetAdequacy <= 0 {
		p.TargetAdequacy = 0.8
	}
	if p.TargetAdequacy > 1 {
		p.TargetAdequacy = 1
	}
	if p.FixtureWattage <= 0 {
		p.FixtureWattage = 100 // 100 W LED street luminaire
	}
	if p.HoursPerNight <= 0 {
		p.HoursPerNight = 11 // ~dusk to dawn average
	}
	if p.EnergyCostPerKwh <= 0 {
		p.EnergyCostPerKwh = 3.5 // TRY/kWh approximate
	}
	if p.CO2KgPerKwh <= 0 {
		p.CO2KgPerKwh = 0.42 // TR grid average kg CO2 / kWh
	}
}

// ScenarioResult is the simulated outcome of applying a policy to the network.
type ScenarioResult struct {
	SegmentsTotal    int `json:"segments_total"`
	SegmentsUpgraded int `json:"segments_upgraded"`
	SegmentsDimmed   int `json:"segments_dimmed"`

	BaselineFixtures int `json:"baseline_fixtures"`
	ProposedFixtures int `json:"proposed_fixtures"`
	AddedFixtures    int `json:"added_fixtures"`

	BaselineAvgRisk      float64 `json:"baseline_avg_risk"`
	ProposedAvgRisk      float64 `json:"proposed_avg_risk"`
	RiskReductionPercent float64 `json:"risk_reduction_percent"`

	BaselineEnergyKwhYear float64 `json:"baseline_energy_kwh_year"`
	ProposedEnergyKwhYear float64 `json:"proposed_energy_kwh_year"`
	EnergyDeltaKwhYear    float64 `json:"energy_delta_kwh_year"` // negative = saving
	EnergySavedPercent    float64 `json:"energy_saved_percent"`

	AnnualCostDelta float64 `json:"annual_cost_delta"` // negative = saving
	CO2DeltaKgYear  float64 `json:"co2_delta_kg_year"` // negative = saving
}

// dimNightShare is the fraction of the night dimming is applied for (late night).
const dimNightShare = 0.5

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func segmentEnergyKwhYear(fixtures int, wattage, hours float64) float64 {
	return float64(fixtures) * wattage * hours * 365.0 / 1000.0
}

// SimulateScenario applies the policy to the given segments and returns the
// aggregated baseline-vs-proposed outcome. Pure function: no side effects.
func SimulateScenario(segments []*StreetSegment, params ScenarioParams) ScenarioResult {
	params.Normalize()

	res := ScenarioResult{SegmentsTotal: len(segments)}
	var baselineRiskSum, proposedRiskSum float64

	for _, s := range segments {
		baseFixtures := s.StreetLightCount
		baseEnergy := segmentEnergyKwhYear(baseFixtures, params.FixtureWattage, params.HoursPerNight)
		res.BaselineFixtures += baseFixtures
		res.BaselineEnergyKwhYear += baseEnergy
		baselineRiskSum += s.RiskScore

		proposedFixtures := baseFixtures
		proposedEnergy := baseEnergy
		proposedRisk := s.RiskScore

		if contains(params.TargetRiskLevels, s.RiskLevel) {
			// Upgrade: add fixtures to reach target adequacy.
			recommended := RecommendedDensityFor(s.RoadType)
			neededDensity := params.TargetAdequacy * recommended
			neededFixtures := int(math.Ceil(neededDensity * (s.LengthM / 100.0)))
			if neededFixtures > proposedFixtures {
				proposedFixtures = neededFixtures
				res.SegmentsUpgraded++
			}
			proposedEnergy = segmentEnergyKwhYear(proposedFixtures, params.FixtureWattage, params.HoursPerNight)
			bd := ScoreSegment(proposedFixtures, s.LengthM, s.RoadType, s.NightSampleRatio)
			proposedRisk = bd.RiskScore
		} else if params.NightDimmingPercent > 0 && s.RiskLevel == string(RiskLow) {
			// Dim well-lit, low-risk segments late at night to save energy.
			factor := 1.0 - (params.NightDimmingPercent/100.0)*dimNightShare
			proposedEnergy = baseEnergy * factor
			res.SegmentsDimmed++
			// Risk unchanged: dimming only where adequacy is already high.
		}

		res.ProposedFixtures += proposedFixtures
		res.ProposedEnergyKwhYear += proposedEnergy
		proposedRiskSum += proposedRisk
	}

	n := float64(max(1, len(segments)))
	res.AddedFixtures = res.ProposedFixtures - res.BaselineFixtures
	res.BaselineAvgRisk = round1(baselineRiskSum / n)
	res.ProposedAvgRisk = round1(proposedRiskSum / n)
	if res.BaselineAvgRisk > 0 {
		res.RiskReductionPercent = round1((res.BaselineAvgRisk - res.ProposedAvgRisk) / res.BaselineAvgRisk * 100.0)
	}

	res.EnergyDeltaKwhYear = round1(res.ProposedEnergyKwhYear - res.BaselineEnergyKwhYear)
	if res.BaselineEnergyKwhYear > 0 {
		res.EnergySavedPercent = round1(-res.EnergyDeltaKwhYear / res.BaselineEnergyKwhYear * 100.0)
	}
	res.AnnualCostDelta = round1(res.EnergyDeltaKwhYear * params.EnergyCostPerKwh)
	res.CO2DeltaKgYear = round1(res.EnergyDeltaKwhYear * params.CO2KgPerKwh)
	res.BaselineEnergyKwhYear = round1(res.BaselineEnergyKwhYear)
	res.ProposedEnergyKwhYear = round1(res.ProposedEnergyKwhYear)

	return res
}
