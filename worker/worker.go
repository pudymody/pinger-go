package worker

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/pudymody/pinger-go/endpoint"
	"github.com/pudymody/pinger-go/hit"

	"github.com/go-co-op/gocron/v2"
)

type EndpointService interface {
	Get(ctx context.Context, id int64) (endpoint.Endpoint, error)
	GetAll(ctx context.Context) ([]endpoint.Endpoint, error)
}

type HitService interface {
	Insert(ctx context.Context, item hit.Hit) error
}

type Logger interface {
	ErrorContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
}

type Worker struct {
	endpointService EndpointService
	hitService      HitService
	logger          Logger
	scheduler       gocron.Scheduler
}

func NewWorker(endpointService EndpointService, hitService HitService, logger Logger) Worker {
	return Worker{
		endpointService: endpointService,
		hitService:      hitService,
		logger:          logger,
	}
}

func (w *Worker) work(ctx context.Context, id int64) {
	endpoint, err := w.endpointService.Get(ctx, id)
	if err != nil {
		w.logger.ErrorContext(ctx, "Getting endpoint", "id", id, "error", err)
		return
	}

	req, err := http.NewRequest("GET", endpoint.Domain, nil)
	if err != nil {
		w.logger.ErrorContext(ctx, "Creating endpoint request", "id", id, "domain", endpoint.Domain, "error", err)
		return
	}

	var start time.Time
	var latency time.Duration

	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			latency = time.Since(start)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
	client := &http.Client{
		Timeout: endpoint.Timeout,
	}
	resp, err := client.Do(req)
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	var timeoutErr net.Error
	isTimeout := errors.As(err, &timeoutErr) && timeoutErr.Timeout()
	if err != nil && !isTimeout {
		w.logger.ErrorContext(ctx, "Requesting domain", "id", id, "domain", endpoint.Domain, "error", err)
		return
	}

	status := hit.StatusUP
	if err != nil || resp.StatusCode != endpoint.CodeOK {
		status = hit.StatusDOWN
		w.logger.InfoContext(ctx, "Check returned down", "endpoint_id", endpoint.ID, "status", resp.StatusCode, "error", err)
	}
	insertErr := w.hitService.Insert(ctx, hit.Hit{
		EndpointID: endpoint.ID,
		Latency:    latency.Milliseconds(),
		Status:     status,
		CreatedAt:  time.Now().UTC(),
	})
	if insertErr != nil {
		w.logger.ErrorContext(ctx, "Inserting hit", "id", id, "endpoint", endpoint, "error", insertErr)
	}
}

func (w *Worker) Start(ctx context.Context) error {
	s, err := gocron.NewScheduler()
	if err != nil {
		return err
	}
	w.scheduler = s

	endpoints, err := w.endpointService.GetAll(ctx)
	if err != nil {
		return err
	}

	for _, e := range endpoints {
		_, err := s.NewJob(
			gocron.DurationJob(e.Interval),
			gocron.NewTask(w.work, e.ID),
		)

		if err != nil {
			return err
		}
		w.logger.InfoContext(ctx, "Added job for endpoint", "endpoint_id", e.ID)
	}
	s.Start()

	w.logger.InfoContext(ctx, "Worker started")
	return nil
}

func (w *Worker) Shutdown(ctx context.Context) error {
	return w.scheduler.Shutdown()
}
