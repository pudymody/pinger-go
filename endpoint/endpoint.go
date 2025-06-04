package endpoint

import (
	"time"
)

type Endpoint struct {
	ID       int64
	Name     string
	Domain   string
	CodeOK   int
	Timeout  time.Duration
	Interval time.Duration
}
