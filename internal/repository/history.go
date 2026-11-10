package repository

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tripgo.local/internal/transaction"
)

type HistoryRepository struct {
	pool *pgxpool.Pool
	sb   squirrel.StatementBuilderType
}

func NewHistoryRepository(pool *pgxpool.Pool) *HistoryRepository {
	return &HistoryRepository{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

type historyQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *HistoryRepository) getQuerier(ctx context.Context) historyQuerier {
	if tx, ok := transaction.TxFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func (r *HistoryRepository) Add(ctx context.Context, tripID uuid.UUID, status string) error {
	q := r.sb.Insert("trip_status_history").
		Columns("trip_id", "status").
		Values(tripID, status)

	sql, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build insert history: %w", err)
	}

	_, err = r.getQuerier(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}
