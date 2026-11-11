package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const jobColumns = `id, idempotency_key, request_hash, payload, state, attempt,
	worker_id, lease_expires_at, created_at, updated_at`

type JobsRepository struct {
	db *pgxpool.Pool
}

func NewJobsRepository(db *pgxpool.Pool) *JobsRepository {
	return &JobsRepository{db: db}
}

func scanJob(row pgx.Row) (Job, error) {
	var j Job
	err := row.Scan(&j.ID, &j.IdempotencyKey, &j.RequestHash, &j.Payload, &j.State,
		&j.Attempt, &j.WorkerID, &j.LeaseExpiresAt, &j.CreatedAt, &j.UpdatedAt)
	return j, err
}

// CreateJob inserts a job or returns the existing one with the same idempotency key.
// created=false means the key already existed; the caller must compare RequestHash.
//
// INSERT ... ON CONFLICT DO NOTHING is race-free: a concurrent insert with the same
// key blocks on the unique index until the first transaction commits, then does nothing.
func (r *JobsRepository) CreateJob(ctx context.Context, key, hash string, payload json.RawMessage) (Job, bool, error) {
	job, err := scanJob(r.db.QueryRow(ctx, `
		INSERT INTO jobs (idempotency_key, request_hash, payload)
		VALUES ($1, $2, $3)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING `+jobColumns, key, hash, payload))
	if err == nil {
		return job, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Job{}, false, fmt.Errorf("insert job: %w", err)
	}

	// Key already taken. This is a separate statement, so under READ COMMITTED it
	// sees the row committed by the concurrent winner.
	job, err = scanJob(r.db.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE idempotency_key = $1`, key))
	if err != nil {
		return Job{}, false, fmt.Errorf("select existing job: %w", err)
	}
	return job, false, nil
}

func (r *JobsRepository) GetJob(ctx context.Context, id int64) (Job, error) {
	job, err := scanJob(r.db.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, fmt.Errorf("get job: %w", err)
	}
	return job, nil
}
