package hardware

import "math"

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
