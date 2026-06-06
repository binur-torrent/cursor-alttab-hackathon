"""Lighting assessment scoring (v2).

Single source of truth for the scoring model on the Python side. The Go backend
re-implements the identical formula in
`internal/domain/lighting/model/scoring.go` so that precomputed seed scores and
live-computed scores always agree. See .cursor/rules/scoring-consistency.mdc.

From environmental features detected in street-level imagery we derive three
headline 0..100 scores and a composite "overall" lighting score:

    adequacy   = clamp(density * brightness / recommended, 0, 1)
    LSS        = effective_adequacy * 100 * night_factor   # lamp coverage
    OCC        = vegetation + tree + building blocking      # higher = worse
    IAS        = adequacy + sidewalk + pole support         # physical infra
    overall    = 0.50*LSS + 0.30*(100 - OCC) + 0.20*IAS
    risk_score = 100 - overall

Every score is a transparent function of detected, urban-only features. The
pipeline never detects or profiles people/vehicles (KVKK).
"""

from __future__ import annotations

import math
from dataclasses import dataclass, field

# Target streetlight fixtures per 100 m by road class.
RECOMMENDED_DENSITY = {
    "highway": 4.0,
    "primary": 3.0,
    "secondary": 2.5,
    "residential": 1.8,
    "service": 1.2,
}

# Retained for compatibility with the network-simulation energy model.
ROAD_WEIGHT = {
    "highway": 1.25,
    "primary": 1.15,
    "secondary": 1.05,
    "residential": 0.95,
    "service": 0.85,
}

# Typical carriageway width (m) by road class. Wider roads demand more light.
ROAD_WIDTH = {
    "highway": 20.0,
    "primary": 16.0,
    "secondary": 12.0,
    "residential": 8.0,
    "service": 6.0,
}

DEFAULT_ROAD_TYPE = "secondary"

WEIGHT_SUFFICIENCY = 0.50
WEIGHT_CLEARNESS = 0.30  # applied to (100 - occlusion)
WEIGHT_INFRA = 0.20


def clamp(value: float, low: float, high: float) -> float:
    return max(low, min(high, value))


def _round(value: float, ndigits: int) -> float:
    """Round half away from zero, matching Go's math.Round (values are >= 0)."""
    factor = 10 ** ndigits
    return math.floor(value * factor + 0.5) / factor


def lighting_density(streetlights: int, length_m: float) -> float:
    if length_m <= 0:
        return 0.0
    return streetlights / (length_m / 100.0)


def recommended_density(road_type: str) -> float:
    return RECOMMENDED_DENSITY.get(road_type, RECOMMENDED_DENSITY[DEFAULT_ROAD_TYPE])


def road_width(road_type: str) -> float:
    return ROAD_WIDTH.get(road_type, ROAD_WIDTH[DEFAULT_ROAD_TYPE])


@dataclass
class Features:
    streetlights: int = 0
    pole_count: int = 0
    length_m: float = 100.0
    road_type: str = DEFAULT_ROAD_TYPE
    night_ratio: float = 0.0
    road_width_m: float = 0.0
    tree_count: int = 0
    vegetation_ratio: float = 0.0
    building_ratio: float = 0.0
    sidewalk_ratio: float = 0.0
    sky_ratio: float = 0.0
    brightness_factor: float = 1.0


@dataclass
class RiskBreakdown:
    density: float
    recommended: float
    adequacy: float
    lighting_sufficiency: float
    occlusion: float
    infrastructure_adequacy: float
    overall_score: float
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


def score_env(f: Features) -> RiskBreakdown:
    road_type = f.road_type or DEFAULT_ROAD_TYPE
    length_m = f.length_m if f.length_m > 0 else 100.0
    brightness = f.brightness_factor if f.brightness_factor > 0 else 1.0

    density = lighting_density(f.streetlights, length_m)
    recommended = recommended_density(road_type)
    adequacy = clamp(density * brightness / recommended, 0.0, 1.0) if recommended > 0 else 1.0

    # Lighting Sufficiency.
    night_factor = 1.0 - 0.15 * clamp(f.night_ratio, 0.0, 1.0) * (1.0 - adequacy)
    base_width = road_width(road_type)
    width = f.road_width_m if f.road_width_m > 0 else base_width
    width_demand = clamp(width / base_width, 0.7, 1.8)
    effective_adequacy = clamp(adequacy / width_demand, 0.0, 1.0)
    lss = clamp(effective_adequacy * 100 * night_factor, 0.0, 100.0)

    # Occlusion (higher = more blocked).
    occ_raw = (
        0.55 * clamp(f.vegetation_ratio, 0.0, 1.0)
        + 0.25 * clamp(f.tree_count / 8.0, 0.0, 1.0)
        + 0.20 * clamp(f.building_ratio, 0.0, 1.0)
    )
    occ = clamp(occ_raw, 0.0, 1.0) * 100.0

    # Infrastructure Adequacy.
    expected_poles = math.ceil(recommended * (length_m / 100.0))
    pole_support = clamp(f.pole_count / expected_poles, 0.0, 1.0) if expected_poles > 0 else 0.0
    ias_raw = 0.55 * adequacy + 0.25 * clamp(f.sidewalk_ratio, 0.0, 1.0) + 0.20 * pole_support
    ias = clamp(ias_raw, 0.0, 1.0) * 100.0

    overall = clamp(
        WEIGHT_SUFFICIENCY * lss + WEIGHT_CLEARNESS * (100.0 - occ) + WEIGHT_INFRA * ias,
        0.0,
        100.0,
    )
    risk = clamp(100.0 - overall, 0.0, 100.0)

    return RiskBreakdown(
        density=_round(density, 3),
        recommended=recommended,
        adequacy=_round(adequacy, 3),
        lighting_sufficiency=_round(lss, 1),
        occlusion=_round(occ, 1),
        infrastructure_adequacy=_round(ias, 1),
        overall_score=_round(overall, 1),
        risk_score=_round(risk, 1),
        risk_level=risk_level(risk),
    )


def score_segment(
    streetlights: int,
    length_m: float,
    road_type: str,
    night_sample_ratio: float = 0.0,
    **features,
) -> RiskBreakdown:
    """Score a segment. Extra environmental features may be passed by keyword
    (vegetation_ratio, tree_count, building_ratio, road_width_m, sidewalk_ratio,
    sky_ratio, pole_count, brightness_factor)."""
    return score_env(
        Features(
            streetlights=streetlights,
            length_m=length_m,
            road_type=road_type,
            night_ratio=night_sample_ratio,
            **features,
        )
    )
