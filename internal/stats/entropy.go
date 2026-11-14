package stats

import (
	"math"
	"sort"
)

// SummaryStats holds basic descriptive statistics.
type SummaryStats struct {
	Count  int     `json:"count"`
	Mean   float64 `json:"mean"`
	Std    float64 `json:"std"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	P50    float64 `json:"p50"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
}

// ComputeSummaryStats calculates statistical moments and percentiles for a slice of floats.
func ComputeSummaryStats(values []float64) SummaryStats {
	if len(values) == 0 {
		return SummaryStats{}
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	var sum float64
	for _, v := range sorted {
		sum += v
	}
	mean := sum / float64(n)

	var varSum float64
	for _, v := range sorted {
		diff := v - mean
		varSum += diff * diff
	}
	std := 0.0
	if n > 1 {
		std = math.Sqrt(varSum / float64(n-1))
	}

	return SummaryStats{
		Count: n,
		Mean:  math.Round(mean*100) / 100,
		Std:   math.Round(std*100) / 100,
		Min:   sorted[0],
		Max:   sorted[n-1],
		P50:   percentile(sorted, 0.50),
		P95:   percentile(sorted, 0.95),
		P99:   percentile(sorted, 0.99),
	}
}

// ComputeShannonEntropy calculates discrete Shannon entropy H = -\sum p_i \log_2(p_i).
func ComputeShannonEntropy(counts map[string]int) float64 {
	total := 0
	for _, c := range counts {
		total += c
	}
	if total == 0 {
		return 0.0
	}

	var entropy float64
	for _, c := range counts {
		if c <= 0 {
			continue
		}
		p := float64(c) / float64(total)
		entropy -= p * math.Log2(p)
	}

	return math.Round(entropy*1000) / 1000
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0.0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}

	index := p * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	weight := index - float64(lower)

	return sorted[lower]*(1.0-weight) + sorted[upper]*weight
}
