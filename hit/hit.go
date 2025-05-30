package hit

import (
	"time"
)

type Status string
const (
	StatusUP Status = "UP"
	StatusDOWN Status = "DOWN"
)

type Hit struct {
	EndpointID int
	Status Status
	Latency int
	CreatedAt time.Time
}
