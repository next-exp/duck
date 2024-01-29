package duck

// RunStatistics holds statistics about a data processing run.
type RunStatistics struct {
	Host   string `json:"host"`
	Events int64  `json:"events"`
	Bytes  int64  `json:"bytes"`
	Errors int64  `json:"errors"`
}
