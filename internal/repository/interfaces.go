package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/model"
)

type UserStore interface {
	Save(ctx context.Context, u model.User) (uuid.UUID, error)
	GetByLogin(ctx context.Context, login string) (model.User, error)
	GetRefreshVersion(ctx context.Context, userID uuid.UUID) (int, error)
	EnsureRefreshVersion(ctx context.Context, userID uuid.UUID) (int, error)
	IncrementRefreshVersion(ctx context.Context, userID uuid.UUID) (int, error)
}

type ItemStore interface {
	Create(ctx context.Context, rec model.ItemRecord) (model.ItemRecord, error)
	Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error)
	List(ctx context.Context, userID uuid.UUID, filterType *string, search *string, limit int, offset int) (ListResult, error)
	Update(ctx context.Context, userID uuid.UUID, id uuid.UUID, upd ItemUpdateRecord) (model.ItemRecord, error)
	Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
	BeginTx(ctx context.Context) (*sql.Tx, error)
	GetWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) (model.ItemRecord, error)
	DeleteWithTx(ctx context.Context, tx *sql.Tx, userID uuid.UUID, id uuid.UUID) error
}
