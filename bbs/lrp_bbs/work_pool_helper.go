package lrp_bbs

import "github.com/cloudfoundry/gunk/workpool"

func constructWorkPool(numTasks, maxWorkers int) (*workpool.WorkPool, error) {
	var numWorkers int
	if numTasks < maxWorkers {
		numWorkers = numTasks
	} else {
		numWorkers = maxWorkers
	}

	return workpool.New(numWorkers, numTasks-numWorkers)
}
