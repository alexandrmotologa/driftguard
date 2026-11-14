package stats

import (
	"math"
	"sort"

	"github.com/alexandrmotologa/driftguard/internal/model"
)

const (
	StateStart = "__START__"
	StateEnd   = "__END__"
)

// BuildMarkovMatrix computes the empirical first-order Markov transition probability matrix
// from a slice of agent traces.
func BuildMarkovMatrix(traces []model.Trace, monitoredTools []string, smoothingAlpha float64) model.MarkovMatrix {
	if smoothingAlpha < 0 {
		smoothingAlpha = 0.01
	}

	stateSet := make(map[string]struct{})
	stateSet[StateStart] = struct{}{}
	stateSet[StateEnd] = struct{}{}

	for _, t := range monitoredTools {
		stateSet[t] = struct{}{}
	}

	// Also discover any unknown/unmonitored tools invoked in actual traces
	for _, tr := range traces {
		for _, s := range tr.Steps {
			if s.ToolName != "" {
				stateSet[s.ToolName] = struct{}{}
			}
		}
	}

	states := make([]string, 0, len(stateSet))
	for s := range stateSet {
		states = append(states, s)
	}
	sort.Strings(states)

	// Initialize raw transition counts
	rawCounts := make(map[string]map[string]int)
	for _, from := range states {
		rawCounts[from] = make(map[string]int)
		for _, to := range states {
			rawCounts[from][to] = 0
		}
	}

	totalTransitions := 0

	for _, tr := range traces {
		if len(tr.Steps) == 0 {
			// Direct start -> end transition
			rawCounts[StateStart][StateEnd]++
			totalTransitions++
			continue
		}

		// Transition 1: __START__ -> first tool
		firstTool := tr.Steps[0].ToolName
		if firstTool != "" {
			rawCounts[StateStart][firstTool]++
			totalTransitions++
		}

		// Middle transitions: tool_i -> tool_{i+1}
		for i := 0; i < len(tr.Steps)-1; i++ {
			from := tr.Steps[i].ToolName
			to := tr.Steps[i+1].ToolName
			if from != "" && to != "" {
				rawCounts[from][to]++
				totalTransitions++
			}
		}

		// Final transition: last tool -> __END__
		lastTool := tr.Steps[len(tr.Steps)-1].ToolName
		if lastTool != "" {
			rawCounts[lastTool][StateEnd]++
			totalTransitions++
		}
	}

	// Compute smoothed transition probabilities
	k := float64(len(states))
	probabilities := make(map[string]map[string]float64)

	for _, from := range states {
		probabilities[from] = make(map[string]float64)

		rowSum := 0
		for _, to := range states {
			rowSum += rawCounts[from][to]
		}

		for _, to := range states {
			count := float64(rawCounts[from][to])
			if rowSum == 0 {
				// Uniform prior if state never observed
				probabilities[from][to] = 1.0 / k
			} else {
				// Laplace smoothing
				prob := (count + smoothingAlpha) / (float64(rowSum) + smoothingAlpha*k)
				probabilities[from][to] = math.Round(prob*10000) / 10000
			}
		}
	}

	return model.MarkovMatrix{
		States:           states,
		Probabilities:    probabilities,
		RawCounts:        rawCounts,
		TotalTransitions: totalTransitions,
	}
}

// CompareMarkovMatrices calculates the perturbation and divergence between baseline and current matrices.
func CompareMarkovMatrices(base, curr model.MarkovMatrix, loopThreshold float64) (model.MatrixPerturbation, []model.LoopAnomaly) {
	stateSet := make(map[string]struct{})
	for _, s := range base.States {
		stateSet[s] = struct{}{}
	}
	for _, s := range curr.States {
		stateSet[s] = struct{}{}
	}

	allStates := make([]string, 0, len(stateSet))
	for s := range stateSet {
		allStates = append(allStates, s)
	}
	sort.Strings(allStates)

	var totalAbsDelta float64
	var maxEdgeDeviation float64
	var maxEdgeFrom, maxEdgeTo string

	var novelTransitions []model.TransitionEdge
	var perturbedEdges []model.PerturbedEdge
	var loopAnomalies []model.LoopAnomaly

	for _, from := range allStates {
		for _, to := range allStates {
			var pBase, pCurr float64
			if base.Probabilities != nil && base.Probabilities[from] != nil {
				pBase = base.Probabilities[from][to]
			}
			if curr.Probabilities != nil && curr.Probabilities[from] != nil {
				pCurr = curr.Probabilities[from][to]
			}

			count := 0
			if curr.RawCounts != nil && curr.RawCounts[from] != nil {
				count = curr.RawCounts[from][to]
			}

			delta := math.Abs(pCurr - pBase)
			totalAbsDelta += delta

			if delta > maxEdgeDeviation {
				maxEdgeDeviation = delta
				maxEdgeFrom = from
				maxEdgeTo = to
			}

			// Flag novel transitions (unseen in baseline with non-trivial frequency in current window)
			baseRawCount := 0
			if base.RawCounts != nil && base.RawCounts[from] != nil {
				baseRawCount = base.RawCounts[from][to]
			}

			if baseRawCount == 0 && count > 0 {
				novelTransitions = append(novelTransitions, model.TransitionEdge{
					From:         from,
					To:           to,
					CurrentProb:  math.Round(pCurr*1000) / 1000,
					BaselineProb: 0,
					Count:        count,
				})
			}

			if delta >= 0.08 {
				perturbedEdges = append(perturbedEdges, model.PerturbedEdge{
					From:         from,
					To:           to,
					Delta:        math.Round(delta*1000) / 1000,
					BaselineProb: math.Round(pBase*1000) / 1000,
					CurrentProb:  math.Round(pCurr*1000) / 1000,
				})
			}

			// Detect self-loop anomalies
			if from == to && from != StateStart && from != StateEnd {
				if loopThreshold <= 0 {
					loopThreshold = 0.20
				}
				if pCurr >= loopThreshold && count > 1 {
					loopAnomalies = append(loopAnomalies, model.LoopAnomaly{
						ToolName:    from,
						Probability: math.Round(pCurr*1000) / 1000,
						Occurrences: count,
					})
				}
			}
		}
	}

	numPairs := float64(len(allStates) * len(allStates))
	meanAbsDev := 0.0
	if numPairs > 0 {
		meanAbsDev = totalAbsDelta / numPairs
	}

	// Sort perturbed edges descending by delta
	sort.Slice(perturbedEdges, func(i, j int) bool {
		return perturbedEdges[i].Delta > perturbedEdges[j].Delta
	})

	perturbation := model.MatrixPerturbation{
		MeanAbsDeviation: math.Round(meanAbsDev*10000) / 10000,
		MaxEdgeDeviation: math.Round(maxEdgeDeviation*10000) / 10000,
		MaxEdgeFrom:      maxEdgeFrom,
		MaxEdgeTo:        maxEdgeTo,
		NovelTransitions: novelTransitions,
		PerturbedEdges:   perturbedEdges,
	}

	return perturbation, loopAnomalies
}
