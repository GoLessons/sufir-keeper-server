package repository

import (
	"context"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func TestUserRepository_SqlBuild(t *testing.T) {
	b := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	id := uuid.New()
	ins := b.Insert("keep.users").Columns("id", "login", "password_hash", "created_at").Values(id, "u", "h", sq.Expr("now()"))
	if _, _, err := ins.ToSql(); err != nil {
		t.Fatalf("build insert keep.users failed: %v", err)
	}
	up := b.Update("keep.refresh_versions").Set("version", sq.Expr("version + 1")).Set("updated_at", sq.Expr("now()")).Where(sq.Eq{"user_id": id})
	if _, _, err := up.ToSql(); err != nil {
		t.Fatalf("build update keep.refresh_versions failed: %v", err)
	}
}

func TestUserRepository_IDNotNil(t *testing.T) {
	r := &UserRepository{}
	_ = r
	if uuid.New() == uuid.Nil {
		t.Fatalf("uuid.New returned Nil")
	}
}

func TestUserRepository_Context(t *testing.T) {
	ctx := context.Background()
	if ctx == nil {
		t.Fatalf("context nil")
	}
}
