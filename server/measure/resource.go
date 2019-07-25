////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package measure

import (
	"sync"
	"time"
)

// A metric for memory and thread usage
type ResourceMetric struct {
	Time                      time.Time
	MemoryAllocated           string
	MemoryAllocationThreshold int64
	NumThreads                int
	HighestMemThreads         string
}

// Contains a mutable resource metric accessed through a mutex
type ResourceMonitor struct {
	lastMetric *ResourceMetric
	sync.Mutex
}

// Get a resource metric using a lock
func (resMon ResourceMonitor) Get() *ResourceMetric {
	resMon.Lock()
	defer resMon.Unlock()
	lastMetric := resMon.lastMetric
	// copy into new object
	rm := ResourceMetric{
		Time:                      lastMetric.Time,
		MemoryAllocated:           lastMetric.MemoryAllocated,
		MemoryAllocationThreshold: lastMetric.MemoryAllocationThreshold,
		NumThreads:                lastMetric.NumThreads,
		HighestMemThreads:         lastMetric.HighestMemThreads,
	}
	return &rm
}

// Set a resource metric using a lock
func (resMon *ResourceMonitor) Set(rm *ResourceMetric) {
	resMon.Lock()
	defer resMon.Unlock()
	resMon.lastMetric = rm
}
