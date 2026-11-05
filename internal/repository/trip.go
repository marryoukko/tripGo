package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tripgo.local/internal/transaction"
)

var (
	ErrDriverBusy  = errors.New("driver already has an active trip")
	ErrNotFound    = errors.New("trip not found")
	ErrAlreadyDone = errors.New("trip already completed")
)

type Trip struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartLat   float64
	StartLon   float64
	EndLat     float64
	EndLon     float64
	Price      int64
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
}

type TripRepository struct {
	pool *pgxpool.Pool
	sb   squirrel.StatementBuilderType
}

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *TripRepository) getQuerier(ctx context.Context) querier {
	if tx, ok := transaction.TxFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func (r *TripRepository) Create(ctx context.Context, t *Trip) error {
	q := r.sb.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_lat", "start_lon",
			"end_lat", "end_lon", "price", "status", "started_at").
		Values(t.ID, t.UserID, t.DriverID, t.StartLat, t.StartLon,
			t.EndLat, t.EndLon, t.Price, t.Status, t.StartedAt)

	sql, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build insert: %w", err)
	}

	_, err = r.getQuerier(ctx).Exec(ctx, sql, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *TripRepository) GetByID(ctx context.Context, id uuid.UUID) (*Trip, error) {
	q := r.sb.Select(
		"id", "user_id", "driver_id", "start_lat", "start_lon",
		"end_lat", "end_lon", "price", "status", "started_at", "finished_at",
	).From("trips").Where(squirrel.Eq{"id": id})

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select: %w", err)
	}

	t := &Trip{}
	err = r.getQuerier(ctx).QueryRow(ctx, sql, args...).Scan(
		&t.ID, &t.UserID, &t.DriverID, &t.StartLat, &t.StartLon,
		&t.EndLat, &t.EndLon, &t.Price, &t.Status, &t.StartedAt, &t.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select trip: %w", err)
	}
	return t, nil
}

func (r *TripRepository) Finish(ctx context.Context, id uuid.UUID) error {
	q := r.sb.Update("trips").
		Set("status", "completed").
		Set("finished_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": id, "status": "active"})

	sql, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build update: %w", err)
	}

	tag, err := r.getQuerier(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	if tag.RowsAffected() == 0 {
		t, err := r.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if t.Status == "completed" {
			return ErrAlreadyDone
		}
	}
	return nil
}
