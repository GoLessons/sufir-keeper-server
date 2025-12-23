package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

type Client struct {
	SQL     *sql.DB
	Builder squirrel.StatementBuilderType
}

type Options struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func NewClient(parentContext context.Context, dataSourceName string, options Options) (*Client, error) {
	configuration, err := pgx.ParseConfig(dataSourceName)
	if err != nil {
		return nil, err
	}
	sqlDatabase := stdlib.OpenDB(*configuration)
	pingContext, cancel := context.WithTimeout(parentContext, 5*time.Second)
	defer cancel()
	if err := sqlDatabase.PingContext(pingContext); err != nil {
		_ = sqlDatabase.Close()
		return nil, err
	}

	if options.MaxOpenConns > 0 {
		sqlDatabase.SetMaxOpenConns(options.MaxOpenConns)
	}
	if options.MaxIdleConns > 0 {
		sqlDatabase.SetMaxIdleConns(options.MaxIdleConns)
	}
	if options.ConnMaxLifetime > 0 {
		sqlDatabase.SetConnMaxLifetime(options.ConnMaxLifetime)
	}
	if options.ConnMaxIdleTime > 0 {
		sqlDatabase.SetConnMaxIdleTime(options.ConnMaxIdleTime)
	}

	builder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	return &Client{SQL: sqlDatabase, Builder: builder}, nil
}

func (client *Client) Close() (err error) {
	if client == nil || client.SQL == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			err = nil
		}
	}()
	return client.SQL.Close()
}
