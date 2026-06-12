package postgres

import (
	"context"
	"errors"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type postgresUserStore struct {
	s *postgresStore
}

func (s *postgresUserStore) Create(u *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.s.pool.Exec(context.Background(), query,
		u.ID, u.Email, u.PasswordHash, u.Name, u.Role, u.CreatedAt, u.UpdatedAt,
	)
	return err
}

func (s *postgresUserStore) GetByID(id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u model.User
	err := s.s.pool.QueryRow(context.Background(), query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *postgresUserStore) GetByEmail(email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var u model.User
	err := s.s.pool.QueryRow(context.Background(), query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *postgresUserStore) List() ([]*model.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, created_at, updated_at
		FROM users
		ORDER BY created_at ASC
	`
	rows, err := s.s.pool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (s *postgresUserStore) Update(u *model.User) error {
	query := `
		UPDATE users
		SET email = $2, password_hash = $3, name = $4, role = $5, updated_at = $6
		WHERE id = $1
	`
	res, err := s.s.pool.Exec(context.Background(), query,
		u.ID, u.Email, u.PasswordHash, u.Name, u.Role, u.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *postgresUserStore) CheckProjectAccess(userID, projectID uuid.UUID) bool {
	// This would likely involve a junction table like project_members.
	// For now, we can check if the user is an admin or if there's a record in a members table.
	// Since the migration doesn't have a members table yet, we'll check if the user is an admin.
	
	user, err := s.GetByID(userID)
	if err != nil {
		return false
	}
	if user.Role == model.RoleAdmin {
		return true
	}

	// Fallback to memory check if we had a many-to-many relationship in the model but not in DB yet.
	// Actually, the model has []string Teams and []string Projects.
	// In Postgres, this should be a separate table or a JSONB field.
	// Given the current schema, we don't have a members table.
	
	return false
}
