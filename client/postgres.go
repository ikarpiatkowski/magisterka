package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postgres struct {
	dbpool *pgxpool.Pool
	config *Config
	context context.Context
}

func NewPostgres(ctx context.Context, c *Config) *postgres {
	pg := postgres{
		config:  c,
		context: ctx,
	}
	pg.pgConnect(ctx)

	return &pg
}

func (pg *postgres) pgConnect(ctx context.Context) {
	url := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?pool_max_conns=%d",
		pg.config.Postgres.User, pg.config.Postgres.Password, pg.config.Postgres.Host, pg.config.Postgres.Database, pg.config.Postgres.MaxConnections)

	dbpool, err := pgxpool.New(ctx, url)
	fail(err, "Unable to create connection pool")

	pg.dbpool = dbpool
}
