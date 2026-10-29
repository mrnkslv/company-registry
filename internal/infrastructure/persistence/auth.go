package persistence

import (
	"company-registry/internal/domain/entity"
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type User struct {
	db *sqlx.DB
}

func NewUser(
	db *sqlx.DB,
) User {
	return User{
		db: db,
	}
}

func (u User) GetUser(ctx context.Context, username, passwordHash string) (*entity.User, error) {
	const query = `
		SELECT id, name, username, password_hash
		FROM users
		WHERE username = $1 AND password_hash = $2`

	var user entity.User
	if err := u.db.GetContext(ctx, &user, query, username, passwordHash); err != nil {
		return nil, fmt.Errorf("db.GetContext: %w", mapError(err))
	}

	return &user, nil
}
