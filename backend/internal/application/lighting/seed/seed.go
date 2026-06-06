// Package seed embeds the anonymized AI-pipeline output so the backend can
// auto-load demo data on first start (e.g. on Render free tier) without a
// separate ingestion step. Regenerate with:
//
//	cd ai && python -m pipeline.generate_seed --out ../data/seed/segments.json
//	cp ../data/seed/segments.json backend/internal/application/lighting/seed/segments.json
//
// The file contains derived metadata only - never raw imagery (KVKK compliant).
package seed

import _ "embed"

//go:embed segments.json
var SegmentsJSON []byte
