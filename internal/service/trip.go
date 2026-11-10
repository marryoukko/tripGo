package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"tripgo.local/internal/repository"
	"tripgo.local/internal/transaction"
)

type Trip = repository.Trip

type CreateTripInput struct {
	UserID   uuid.UUID
	DriverID uuid.UUID
	StartLat float64
	StartLon float64
	EndLat   float64
	EndLon   float64
	Price    int64
}

type TripService struct {
	tripRepo    *repository.TripRepository
	historyRepo *repository.HistoryRepository
	txManager   transaction.TxManager
}

func NewTripService(
	tripRepo *repository.TripRepository,
	historyRepo *repository.HistoryRepository,
	txManager transaction.TxManager,
) *TripService {
	return &TripService{
		tripRepo:    tripRepo,
		historyRepo: historyRepo,
		txManager:   txManager,
	}
}

func (s *TripService) Create(ctx context.Context, in CreateTripInput) (*Trip, error) {
	trip := &Trip{
		ID:        uuid.New(),
		UserID:    in.UserID,
		DriverID:  in.DriverID,
		StartLat:  in.StartLat,
		StartLon:  in.StartLon,
		EndLat:    in.EndLat,
		EndLon:    in.EndLon,
		Price:     in.Price,
		Status:    "active",
		StartedAt: time.Now().UTC(),
	}

	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		if err := s.tripRepo.Create(ctx, trip); err != nil {
			return err
		}
		return s.historyRepo.Add(ctx, trip.ID, "active")
	})
	if err != nil {
		return nil, err
	}
	return trip, nil
}

func (s *TripService) Get(ctx context.Context, id uuid.UUID) (*Trip, error) {
	return s.tripRepo.GetByID(ctx, id)
}

func (s *TripService) Finish(ctx context.Context, id uuid.UUID) (*Trip, error) {
	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		return s.tripRepo.Finish(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return s.tripRepo.GetByID(ctx, id)
}

var (
	ErrNotFound    = repository.ErrNotFound
	ErrAlreadyDone = repository.ErrAlreadyDone
	ErrDriverBusy  = repository.ErrDriverBusy
)

var _ = errors.Is
