"""Geospatial helpers for Istanbul: bounding box + segment binning."""

from __future__ import annotations

import math
from dataclasses import dataclass

# Istanbul bounding box (covers both European and Anatolian sides).
ISTANBUL_BBOX = {
    "min_lat": 40.80,
    "max_lat": 41.30,
    "min_lon": 28.50,
    "max_lon": 29.45,
}


def in_istanbul(lat: float, lon: float) -> bool:
    b = ISTANBUL_BBOX
    return b["min_lat"] <= lat <= b["max_lat"] and b["min_lon"] <= lon <= b["max_lon"]


def haversine_m(lat1: float, lon1: float, lat2: float, lon2: float) -> float:
    """Great-circle distance in meters."""
    r = 6371000.0
    p1, p2 = math.radians(lat1), math.radians(lat2)
    dphi = math.radians(lat2 - lat1)
    dlmb = math.radians(lon2 - lon1)
    a = math.sin(dphi / 2) ** 2 + math.cos(p1) * math.cos(p2) * math.sin(dlmb / 2) ** 2
    return 2 * r * math.asin(math.sqrt(a))


@dataclass(frozen=True)
class GridCell:
    row: int
    col: int


def cell_for(lat: float, lon: float, cell_m: float = 150.0) -> GridCell:
    """Map a coordinate to a ~`cell_m` sized grid cell. Each cell becomes a segment."""
    # Approximate meters-per-degree at Istanbul's latitude.
    m_per_deg_lat = 111_320.0
    m_per_deg_lon = 111_320.0 * math.cos(math.radians(41.0))
    row = int((lat * m_per_deg_lat) // cell_m)
    col = int((lon * m_per_deg_lon) // cell_m)
    return GridCell(row=row, col=col)


def cell_center(cell: GridCell, cell_m: float = 150.0) -> tuple[float, float]:
    m_per_deg_lat = 111_320.0
    m_per_deg_lon = 111_320.0 * math.cos(math.radians(41.0))
    lat = ((cell.row + 0.5) * cell_m) / m_per_deg_lat
    lon = ((cell.col + 0.5) * cell_m) / m_per_deg_lon
    return lat, lon
