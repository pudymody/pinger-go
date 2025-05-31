package endpoint

import (
	"context"
)

type Storage interface {
	InsertEndpoint(ctx context.Context, item Endpoint) error
	GetAllEndpoints(ctx context.Context) ([]Endpoint, error)
	GetEndpoint(ctx context.Context, id int64) (Endpoint, error)
	UpdateEndpoint(ctx context.Context, item Endpoint) error
}

type EndpointService struct {
	storage Storage
}

func NewEndpointService(storage Storage) EndpointService {
	return EndpointService{
		storage: storage,
	}
}

func (s *EndpointService) Insert(ctx context.Context, item Endpoint) error {
	return s.storage.InsertEndpoint(ctx, item)
}

func (s *EndpointService) GetAll(ctx context.Context) ([]Endpoint, error) {
	return s.storage.GetAllEndpoints(ctx)
}

func (s *EndpointService) Get(ctx context.Context, id int64) (Endpoint, error) {
	return s.storage.GetEndpoint(ctx, id)
}

func (s *EndpointService) Update(ctx context.Context, item Endpoint) error {
	return s.storage.UpdateEndpoint(ctx, item)
}
