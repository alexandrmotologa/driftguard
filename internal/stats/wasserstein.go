package stats

import (
	"math"
	"math/rand"
	"sort"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

// ComputeWasserstein1D calculates the first Wasserstein distance (Earth Mover's Distance)
// between two 1D empirical distributions X and Y.
// W_1(P, Q) = \int |F_P(t) - F_Q(t)| dt
func ComputeWasserstein1D(x, y []float64) float64 {
	if len(x) == 0 || len(y) == 0 {
		return 0.0
	}

	// Make copies so we do not mutate inputs
	xSorted := make([]float64, len(x))
	copy(xSorted, x)
	sort.Float64s(xSorted)

	ySorted := make([]float64, len(y))
	copy(ySorted, y)
	sort.Float64s(ySorted)

	// Combine unique split points
	pointsMap := make(map[float64]struct{}, len(x)+len(y))
	for _, val := range xSorted {
		pointsMap[val] = struct{}{}
	}
	for _, val := range ySorted {
		pointsMap[val] = struct{}{}
	}

	uniquePoints := make([]float64, 0, len(pointsMap))
	for pt := range pointsMap {
		uniquePoints = append(uniquePoints, pt)
	}
	sort.Float64s(uniquePoints)

	if len(uniquePoints) < 2 {
		// Single point distribution
		return math.Abs(xSorted[0] - ySorted[0])
	}

	n := float64(len(xSorted))
	m := float64(len(ySorted))

	var totalDistance float64
	xIdx := 0
	yIdx := 0

	for i := 0; i < len(uniquePoints)-1; i++ {
		t := uniquePoints[i]
		nextT := uniquePoints[i+1]
		delta := nextT - t

		for xIdx < len(xSorted) && xSorted[xIdx] <= t {
			xIdx++
		}
		for yIdx < len(ySorted) && ySorted[yIdx] <= t {
			yIdx++
		}

		cdfX := float64(xIdx) / n
		cdfY := float64(yIdx) / m

		totalDistance += delta * math.Abs(cdfX-cdfY)
	}

	return totalDistance
}

// GenerateGaussianSamples produces synthetic samples from a normal distribution N(mean, std^2).
// Used when only mean and standard deviation are provided in configuration.
func GenerateGaussianSamples(mean, std float64, n int) []float64 {
	if n <= 0 {
		n = 100
	}
	if std <= 0 {
		std = 0.5
	}
	rng := rand.New(rand.NewSource(42))
	samples := make([]float64, n)
	for i := 0; i < n; i++ {
		// Box-Muller transform
		u1 := rng.Float64()
		u2 := rng.Float64()
		if u1 <= 0 {
			u1 = 1e-9
		}
		z := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
		val := math.Round(mean + z*std)
		if val < 1 {
			val = 1
		}
		samples[i] = val
	}
	return samples
}

// BuildStepHistogram builds comparable relative frequency bins for baseline vs current step counts.
func BuildStepHistogram(baselineSamples, currentSamples []float64) []model.HistogramBin {
	if len(baselineSamples) == 0 && len(currentSamples) == 0 {
		return nil
	}

	minVal := math.MaxFloat64
	maxVal := -math.MaxFloat64

	for _, v := range baselineSamples {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	for _, v := range currentSamples {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	minStep := int(math.Floor(minVal))
	if minStep < 1 {
		minStep = 1
	}
	maxStep := int(math.Ceil(maxVal))
	if maxStep < minStep {
		maxStep = minStep
	}

	baseCounts := make(map[int]int)
	currCounts := make(map[int]int)

	for _, v := range baselineSamples {
		step := int(math.Round(v))
		baseCounts[step]++
	}
	for _, v := range currentSamples {
		step := int(math.Round(v))
		currCounts[step]++
	}

	nBase := float64(len(baselineSamples))
	nCurr := float64(len(currentSamples))

	bins := make([]model.HistogramBin, 0, maxStep-minStep+1)
	for s := minStep; s <= maxStep; s++ {
		var baseP, currP float64
		if nBase > 0 {
			baseP = float64(baseCounts[s]) / nBase
		}
		if nCurr > 0 {
			currP = float64(currCounts[s]) / nCurr
		}
		bins = append(bins, model.HistogramBin{
			Step:      s,
			BaselineP: math.Round(baseP*1000) / 1000,
			CurrentP:  math.Round(currP*1000) / 1000,
		})
	}

	return bins
}
