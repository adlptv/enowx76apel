package sqlite

import (
	"context"
	"database/sql"

	"github.com/enowdev/enowx/store"
)

type warmupStore struct{ db *sql.DB }

func (s *warmupStore) Insert(ctx context.Context, l store.WarmupLog) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO warmup_logs
		   (account_id, provider, label, ok, outcome, status, request, response, usage, duration_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.AccountID, l.Provider, l.Label, l.OK, l.Outcome, l.Status, l.Request, l.Response, l.Usage, l.DurationMS)
	return err
}

func (s *warmupStore) Recent(ctx context.Context, limit int) ([]store.WarmupLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, account_id, provider, label, ok, outcome, status, request, response, usage, duration_ms, created_at
		 FROM warmup_logs ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.WarmupLog
	for rows.Next() {
		var l store.WarmupLog
		if err := rows.Scan(&l.ID, &l.AccountID, &l.Provider, &l.Label, &l.OK, &l.Outcome, &l.Status,
			&l.Request, &l.Response, &l.Usage, &l.DurationMS, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
