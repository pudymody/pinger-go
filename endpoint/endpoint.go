package endpoint

import (
	"time"
)

type Endpoint struct {
	ID      int
	Domain  string
	CodeOK  int
	Timeout time.Duration
	Interval time.Duration
}
