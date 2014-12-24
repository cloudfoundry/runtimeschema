package lrp_bbs

import "github.com/cloudfoundry-incubator/runtime-schema/models"

type Result struct {
	IndicesToStart []int
	IndicesToStop  []int
}

func (r Result) Empty() bool {
	return len(r.IndicesToStart) == 0 && len(r.IndicesToStop) == 0
}

func Reconcile(numDesired int, actuals models.ActualLRPsByIndex) Result {
	result := Result{}

	for i := 0; i < numDesired; i++ {
		if _, hasIndex := actuals[i]; !hasIndex {
			result.IndicesToStart = append(result.IndicesToStart, i)
		}
	}

	for _, actual := range actuals {
		if actual.Index >= numDesired {
			result.IndicesToStop = append(result.IndicesToStop, actual.Index)
		}
	}

	return result
}
