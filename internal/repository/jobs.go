package repository

import (
	"encoding/json"
	"errors"
	"time"
)

type JobState string

const (
	QUEUED    JobState = "queued"
	RUNNING   JobState = "running"
	SUCCEEDED JobState = "succeeded"
	FAILED    JobState = "failed"
	CANCELLED JobState = "cancelled"
)

var ErrNotFound = errors.New("job not found")

type Job struct {
	ID             int64           `json:"id"`
	IdempotencyKey string          `json:"idempotency_key"`
	RequestHash    string          `json:"-"`
	Payload        json.RawMessage `json:"payload"`
	State          JobState        `json:"state"`
	Attempt        int             `json:"attempt"`
	WorkerID       *string         `json:"worker_id"`
	LeaseExpiresAt *time.Time      `json:"lease_expires_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
