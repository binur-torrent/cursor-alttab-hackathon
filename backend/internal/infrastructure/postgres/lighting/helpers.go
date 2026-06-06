package lighting

import "strconv"

func itoa(i int) string { return strconv.Itoa(i) }

func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
