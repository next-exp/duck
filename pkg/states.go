package duck

import "fmt"

type StateType int

const (
	INITIALIZED StateType = iota
	RUNNING
	STARTING
	STOPPING
	PINGING
)

func (s StateType) String() string {
	switch s {
	case INITIALIZED:
		return "INITIALIZED"
	case RUNNING:
		return "RUNNING"
	case STARTING:
		return "STARTING"
	case STOPPING:
		return "STOPPING"
	case PINGING:
		return "PINGING"
	default:
		return fmt.Sprintf("StateType(%d)", s)
	}
}
