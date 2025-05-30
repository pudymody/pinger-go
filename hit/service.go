package hit

import (
	"context"
	"time"
)

type Storage interface {
	InsertHit(ctx context.Context, item Hit) error
	GetHits(ctx context.Context, endpointID int, from time.Time, to time.Time) ([]Hit, error)
}

type HitService struct {
	storage Storage
}

func NewHitService(storage Storage) HitService {
	return HitService{
		storage: storage,
	}
}

func (s *HitService) Insert(ctx context.Context, item Hit) error {
	return s.storage.InsertHit(ctx, item)
}

func (s *HitService) Get(ctx context.Context, endpointID int, from time.Time, to time.Time) ([]Hit, error) {
	return s.storage.GetHits(ctx, endpointID, from, to)
}
