package hit

import (
	"time"
)

type Status string

const (
	StatusUP   Status = "UP"
	StatusDOWN Status = "DOWN"
)

type Hit struct {
	EndpointID int64
	Status     Status
	Latency    int64
	CreatedAt  time.Time
}
