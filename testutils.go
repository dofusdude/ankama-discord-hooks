package main

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

func testutilGetlastinsertedwebhookid() (uuid.UUID, error) {
	ctx := context.Background()
	conn, err := pgxpool.New(ctx, PostgresUrl)
	defer conn.Close()
	var id uuid.UUID
	var createdAt time.Time
	err = conn.QueryRow(ctx, "SELECT id, created_at FROM webhooks ORDER BY created_at DESC LIMIT 1").Scan(&id, &createdAt)
	return id, err
}

func testutilCleartables() error {
	var err error
	ctx := context.Background()
	conn, err := pgxpool.New(ctx, PostgresUrl)
	defer conn.Close()

	_, err = conn.Exec(ctx, "delete from almanax_mentions")
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, "delete from discord_mentions")
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, "delete from almanax_webhooks")
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, "delete from twitter_webhooks")
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, "delete from rss_webhooks")
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, "delete from subscriptions")
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, "delete from webhooks")
	if err != nil {
		return err
	}

	return err
}
