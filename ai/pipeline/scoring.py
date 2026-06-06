"""Lighting adequacy / risk scoring.

This is the single source of truth for the scoring model on the Python side.
The Go backend re-implements the identical formula in
`internal/domain/lighting/model/scoring.go` so that precomputed seed scores and
live-computed scores always agree.

Model (kept deliberately simple and explainable for the demo):

    density          = streetlights / (length_m / 100)      # fixtures per 100 m
    recommended      = RECOMMENDED_DENSITY[road_type]        # target fixtures / 100 m
    adequacy         = clamp(density / recommended, 0, 1)    # 0..1
    base_risk        = (1 - adequacy) * 100
    risk             = clamp(base_risk * road_weight * night_weight, 0, 100)

`road_weight` raises risk on busy roads (a dark highway is more dangerous than a
dark cul-de-sac). `night_weight` raises risk where most samples were captured at
night yet adequacy is still low (evidence the segment is actually dark in use).
"""

from __future__ import annotations

from dataclasses import dataclass

# Target streetlight fixtures per 100 m by road class (illumination engineering
# rules of thumb, simplified). Higher class road -> needs more / brighter light.
RECOMMENDED_DENSITY = {
    "highway": 4.0,
    "primary": 3.0,
    "secondary": 2.5,
    "residential": 1.8,
    "service": 1.2,
}

# How much an underlit road of each class contributes to risk.
ROAD_WEIGHT = {
    "highway": 1.25,
    "primary": 1.15,
    "secondary": 1.05,
    "residential": 0.95,
    "service": 0.85,
}

DEFAULT_ROAD_TYPE = "secondary"


def clamp(value: float, low: float, high: float) -> float:
    return max(low, min(high, value))


def lighting_density(streetlights: int, length_m: float) -> float:
    """Fixtures per 100 m of segment length."""
    if length_m <= 0:
        return 0.0
    return streetlights / (length_m / 100.0)


def recommended_density(road_type: str) -> float:
    return RECOMMENDED_DENSITY.get(road_type, RECOMMENDED_DENSITY[DEFAULT_ROAD_TYPE])


@dataclass
class RiskBreakdown:
    density: float
    recommended: float
    adequacy: float
    risk_score: float
    risk_level: str


def risk_level(score: float) -> str:
    if score >= 75:
        return "critical"
    if score >= 50:
        return "high"
    if score >= 25:
        return "medium"
    return "low"


def score_segment(
    streetlights: int,
    length_m: float,
    road_type: str,
    night_sample_ratio: float = 0.0,
) -> RiskBreakdown:
    density = lighting_density(streetlights, length_m)
    recommended = recommended_density(road_type)
    adequacy = clamp(density / recommended, 0.0, 1.0) if recommended > 0 else 1.0

    base_risk = (1.0 - adequacy) * 100.0
    road_weight = ROAD_WEIGHT.get(road_type, ROAD_WEIGHT[DEFAULT_ROAD_TYPE])
    # Night weight: up to +20% risk when a segment is mostly observed at night
    # but still inadequately lit.
    night_weight = 1.0 + 0.20 * clamp(night_sample_ratio, 0.0, 1.0) * (1.0 - adequacy)

    risk = clamp(base_risk * road_weight * night_weight, 0.0, 100.0)
    return RiskBreakdown(
        density=round(density, 3),
        recommended=recommended,
        adequacy=round(adequacy, 3),
        risk_score=round(risk, 1),
        risk_level=risk_level(risk),
    )
