package web

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/pudymody/pinger-go/endpoint"
	"github.com/pudymody/pinger-go/hit"
)

var tplAdmin = template.Must(template.ParseFS(templates, "admin.html"))
var tplView = template.Must(template.New("view").Funcs(map[string]any{
	"hitDates": func(hits []hit.Hit) []string {
		dates := make([]string, len(hits))
		for i, h := range hits {
			dates[i] = h.CreatedAt.Format(time.RFC3339)
		}

		return dates
	},
	"hitLatencies": func(hits []hit.Hit) []int64 {
		latencies := make([]int64, len(hits))
		for i, h := range hits {
			latencies[i] = h.Latency
		}

		return latencies
	},
	"hitAnnotations": func(hits []hit.Hit) []map[string]any {
		annotations := make([]map[string]any, 0)
		totalHits := len(hits)
		for i := 0; i < totalHits-1; i++ {
			h := hits[i]
			h2 := hits[i+1]

			if h.Latency == 0 {
				annotations = append(annotations, map[string]any{
					"x":         h.CreatedAt.UnixMilli(),
					"x2":        h2.CreatedAt.UnixMilli(),
					"fillColor": "#D9534F",
				})
			}
		}

		return annotations
	},
}).ParseFS(templates, "view.html"))

type EndpointService interface {
	Insert(ctx context.Context, item endpoint.Endpoint) error
	GetAll(ctx context.Context) ([]endpoint.Endpoint, error)
	Update(ctx context.Context, item endpoint.Endpoint) error
	Get(ctx context.Context, id int64) (endpoint.Endpoint, error)
}

type HitService interface {
	Get(ctx context.Context, endpointID int64, from time.Time, to time.Time) ([]hit.Hit, error)
}

type Logger interface {
	ErrorContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
}

type Server struct {
	basePath        string
	httpServer      *http.Server
	endpointService EndpointService
	hitService      HitService
	logger          Logger
	addr            string
}

func NewServer(addr string, basePath string, endpointService EndpointService, hitService HitService, logger Logger) Server {
	return Server{
		basePath:        basePath,
		endpointService: endpointService,
		hitService:      hitService,
		logger:          logger,
		addr:            addr,
	}
}

func endpointFromRequest(req *http.Request) (endpoint.Endpoint, error) {
	id := 0
	idRaw := req.PathValue("id")
	if idRaw != "" {
		idParsed, err := strconv.Atoi(idRaw)
		if err != nil {
			return endpoint.Endpoint{}, fmt.Errorf("Parsing id value: %w", err)
		}
		id = idParsed
	}

	domain := req.PostFormValue("domain")
	codeOkRaw := req.PostFormValue("code_ok")
	codeOk, err := strconv.Atoi(codeOkRaw)
	if err != nil {
		return endpoint.Endpoint{}, fmt.Errorf("Parsing code_ok value: %w", err)
	}

	timeoutRaw := req.PostFormValue("timeout")
	timeout, err := time.ParseDuration(timeoutRaw)
	if err != nil {
		return endpoint.Endpoint{}, fmt.Errorf("Parsing timeout value: %w", err)
	}

	intervalRaw := req.PostFormValue("interval")
	interval, err := time.ParseDuration(intervalRaw)
	if err != nil {
		return endpoint.Endpoint{}, fmt.Errorf("Parsing interval value: %w", err)
	}

	return endpoint.Endpoint{
		ID:      int64(id),
		Domain:  domain,
		CodeOK:  codeOk,
		Timeout: timeout,
		Interval: interval,
	}, nil
}

func (s *Server) writeErr(ctx context.Context, resp http.ResponseWriter, msg string, err error) {
	s.logger.ErrorContext(ctx, msg, "error", err.Error())
	resp.WriteHeader(http.StatusInternalServerError)
	resp.Write([]byte(err.Error()))
}

type viewEndpointItem struct {
	Endpoint endpoint.Endpoint
	Hits     []hit.Hit
}

type viewEndpointData struct {
	CurrentDay  time.Time
	PreviousDay time.Time
	NextDay     time.Time
	Items       []viewEndpointItem
}

func (s *Server) viewEndpoint(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	t0 := time.Now()
	fromQuery := req.URL.Query().Get("from")
	if fromQuery != "" {
		if parsedFrom, err := time.Parse(time.DateOnly, fromQuery); err == nil {
			t0 = parsedFrom
		}
	}

	from := time.Date(t0.Year(), t0.Month(), t0.Day(), 0, 0, 0, 0, time.Local).UTC()
	to := from.AddDate(0, 0, 1)

	endpoints, err := s.endpointService.GetAll(ctx)
	if err != nil {
		s.writeErr(ctx, resp, "getting all endpoints", err)
		return
	}

	items := make([]viewEndpointItem, 0)
	for _, e := range endpoints {
		hits, err := s.hitService.Get(ctx, e.ID, from, to)
		if err != nil {
			s.writeErr(ctx, resp, "getting single endpoint", err)
			return
		}

		items = append(items, viewEndpointItem{
			Endpoint: e,
			Hits:     hits,
		})
	}

	tplView.ExecuteTemplate(resp, "view.html", viewEndpointData{
		Items:       items,
		CurrentDay:  from,
		NextDay:     from.AddDate(0, 0, 1),
		PreviousDay: from.AddDate(0, 0, -1),
	})
}

func (s *Server) listEndpoints(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	endpoints, err := s.endpointService.GetAll(ctx)
	if err != nil {
		s.writeErr(ctx, resp, "getting all endpoints", err)
		return
	}

	tplAdmin.ExecuteTemplate(resp, "admin.html", endpoints)
}

func (s *Server) updateEndpoint(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	endpointPost, err := endpointFromRequest(req)
	if err != nil {
		s.writeErr(ctx, resp, "getting endpoint from request", err)
		return
	}

	err = s.endpointService.Update(ctx, endpointPost)
	if err != nil {
		s.writeErr(ctx, resp, "updating endpoint", err)
		return
	}
	http.Redirect(resp, req, "/endpoint", http.StatusSeeOther)
}
func (s *Server) insertEndpoint(resp http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	endpointPost, err := endpointFromRequest(req)
	if err != nil {
		s.writeErr(ctx, resp, "getting endpoint from request", err)
		return
	}
	err = s.endpointService.Insert(ctx, endpointPost)
	if err != nil {
		s.writeErr(ctx, resp, "inserting endpoint", err)
		return
	}
	http.Redirect(resp, req, "/endpoint", http.StatusSeeOther)
}

func (s *Server) Start(ctx context.Context) {
	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.FileServerFS(templates))

	mux.HandleFunc("GET /", s.viewEndpoint)
	mux.HandleFunc("GET /endpoint/", s.listEndpoints)
	mux.HandleFunc("POST /endpoint/{id}", s.updateEndpoint)
	mux.HandleFunc("POST /endpoint", s.insertEndpoint)

	s.httpServer = &http.Server{
		Addr:           s.addr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.logger.InfoContext(ctx, "Server started", "address", s.addr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.ErrorContext(ctx, "Listening", "error", err)
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) {
	ctxCancel, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errShutdown := s.httpServer.Shutdown(ctxCancel)
	if errShutdown != nil {
		s.logger.ErrorContext(ctx, "Shutting down", "error", errShutdown)
	}
}
