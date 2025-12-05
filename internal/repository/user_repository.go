package repository

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/db"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
)

type UserRepository struct {
	client *db.Client
}

func NewUserRepository(client *db.Client) *UserRepository { return &UserRepository{client: client} }

func (r *UserRepository) Save(ctx context.Context, u model.User) (uuid.UUID, error) {
	createdAt := u.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	upsertUser := r.client.Builder.
		Insert("keep.users").
		Columns("id", "login", "password_hash", "created_at").
		Values(u.ID, u.Login, u.PasswordHash, createdAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET login = EXCLUDED.login, password_hash = EXCLUDED.password_hash")

	sqlUser, argsUser, err := upsertUser.ToSql()
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := r.client.SQL.ExecContext(ctx, sqlUser, argsUser...); err != nil {
		return uuid.Nil, err
	}

	upsertVersion := r.client.Builder.
		Insert("keep.refresh_versions").
		Columns("user_id", "version", "updated_at").
		Values(u.ID, 1, sq.Expr("now()")).
		Suffix("ON CONFLICT (user_id) DO NOTHING")
	sqlVer, argsVer, err := upsertVersion.ToSql()
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := r.client.SQL.ExecContext(ctx, sqlVer, argsVer...); err != nil {
		return uuid.Nil, err
	}

	return u.ID, nil
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (model.User, error) {
	query := r.client.Builder.
		Select("id", "login", "password_hash", "created_at").
		From("keep.users").
		Where(sq.Eq{"login": login}).
		Limit(1)
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return model.User{}, err
	}

	var rec model.User
	err = r.client.SQL.QueryRowContext(ctx, sqlStr, args...).Scan(&rec.ID, &rec.Login, &rec.PasswordHash, &rec.CreatedAt)
	if err != nil {
		return model.User{}, err
	}

	return rec, nil
}

func (r *UserRepository) GetRefreshVersion(ctx context.Context, userID uuid.UUID) (int, error) {
	query := r.client.Builder.
		Select("version").
		From("keep.refresh_versions").
		Where(sq.Eq{"user_id": userID}).Limit(1)
	sqlStr, args, err := query.ToSql()
	if err != nil {
		return 0, err
	}

	var version int
	err = r.client.SQL.QueryRowContext(ctx, sqlStr, args...).Scan(&version)
	if err != nil {
		return 0, err
	}

	return version, nil
}

func (r *UserRepository) IncrementRefreshVersion(ctx context.Context, userID uuid.UUID) (int, error) {
	update := r.client.Builder.Update("keep.refresh_versions").Set("version", sq.Expr("version + 1")).Set("updated_at", sq.Expr("now()"))
	update = update.Where(sq.Eq{"user_id": userID})
	sqlStr, args, err := update.ToSql()
	if err != nil {
		return 0, err
	}
	if _, err := r.client.SQL.ExecContext(ctx, sqlStr, args...); err != nil {
		return 0, err
	}

	return r.GetRefreshVersion(ctx, userID)
}
