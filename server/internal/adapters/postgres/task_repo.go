// server/internal/adapters/postgres/task_repo.go
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lucasmeneses/world-of-agents/server/internal/domain/task"
)

type TaskRepo struct {
	db *DB
}

func NewTaskRepo(db *DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(ctx context.Context, t *task.Task) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO tasks (id, guild_id, posted_by, title, description, priority, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.GuildID, t.PostedBy, t.Title, t.Description,
		string(t.Priority), string(t.Status), t.CreatedAt,
	)
	return err
}

func (r *TaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	row := r.db.Pool.QueryRow(ctx,
		`SELECT id, guild_id, posted_by, claimed_by, title, description, priority, status, result, created_at, claimed_at, completed_at
		 FROM tasks WHERE id = $1`, id)
	return scanTask(row)
}

func (r *TaskRepo) Update(ctx context.Context, t *task.Task) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE tasks SET status = $2, claimed_by = $3, result = $4, claimed_at = $5, completed_at = $6
		 WHERE id = $1`,
		t.ID, string(t.Status), t.ClaimedBy, t.Result, t.ClaimedAt, t.CompletedAt,
	)
	return err
}

func (r *TaskRepo) ListByGuild(ctx context.Context, guildID uuid.UUID, status *task.Status, limit, offset int) ([]task.Task, error) {
	var rows pgx.Rows
	var err error
	if status != nil {
		rows, err = r.db.Pool.Query(ctx,
			`SELECT id, guild_id, posted_by, claimed_by, title, description, priority, status, result, created_at, claimed_at, completed_at
			 FROM tasks WHERE guild_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
			guildID, string(*status), limit, offset)
	} else {
		rows, err = r.db.Pool.Query(ctx,
			`SELECT id, guild_id, posted_by, claimed_by, title, description, priority, status, result, created_at, claimed_at, completed_at
			 FROM tasks WHERE guild_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			guildID, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		t, err := scanTaskRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, rows.Err()
}

func scanTask(row pgx.Row) (*task.Task, error) {
	var t task.Task
	var status, priority string
	var result *string
	var claimedBy *uuid.UUID
	var claimedAt, completedAt *time.Time

	err := row.Scan(&t.ID, &t.GuildID, &t.PostedBy, &claimedBy,
		&t.Title, &t.Description, &priority, &status,
		&result, &t.CreatedAt, &claimedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	t.Status = task.Status(status)
	t.Priority = task.Priority(priority)
	t.ClaimedBy = claimedBy
	t.ClaimedAt = claimedAt
	t.CompletedAt = completedAt
	if result != nil {
		t.Result = *result
	}
	return &t, nil
}

func scanTaskRows(rows pgx.Rows) (*task.Task, error) {
	var t task.Task
	var status, priority string
	var result *string
	var claimedBy *uuid.UUID
	var claimedAt, completedAt *time.Time

	err := rows.Scan(&t.ID, &t.GuildID, &t.PostedBy, &claimedBy,
		&t.Title, &t.Description, &priority, &status,
		&result, &t.CreatedAt, &claimedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	t.Status = task.Status(status)
	t.Priority = task.Priority(priority)
	t.ClaimedBy = claimedBy
	t.ClaimedAt = claimedAt
	t.CompletedAt = completedAt
	if result != nil {
		t.Result = *result
	}
	return &t, nil
}
