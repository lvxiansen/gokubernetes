package common

import "time"

type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     uint64    `json:"value"`
}

type PodMetrics struct {
	// Most recent measure of CPU usage on all cores in nanoseconds.
	CPUUsage *uint64 `json:"cpuUsage"`
	// Pod memory usage in bytes.
	MemoryUsage *uint64 `json:"memoryUsage"`
	// Timestamped samples of CPUUsage over some short period of history
	CPUUsageHistory []MetricPoint `json:"cpuUsageHistory"`
	// Timestamped samples of pod memory usage over some short period of history
	MemoryUsageHistory []MetricPoint `json:"memoryUsageHistory"`
}
