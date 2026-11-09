package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrNotFound       = errors.New("not found")
)

type User struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Timezone  string `json:"timezone"`
	CreatedAt string `json:"created_at"`
}

type UserWithHash struct {
	User
	PasswordHash string
}

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

var defaultCategories = []struct {
	name string
	kind string
}{
	// Chi (expense)
	{"Ăn uống", "expense"},
	{"Di chuyển", "expense"},
	{"Nhà ở", "expense"},
	{"Hóa đơn", "expense"},
	{"Giải trí", "expense"},
	{"Sức khỏe", "expense"},
	{"Học tập", "expense"},
	{"Khác", "expense"},
	// Thu (income)
	{"Lương", "income"},
	{"Thưởng", "income"},
	{"Khác", "income"},
}

func (r *Repo) CreateUserWithDefaults(ctx context.Context, email, passwordHash string) (*User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	timezone := "Asia/Ho_Chi_Minh"

	res, err := tx.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, timezone, created_at)
		VALUES (?, ?, ?, ?)
	`, email, passwordHash, timezone, now)
	if err != nil {
		if stringsContains(err.Error(), "UNIQUE") {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	for _, cat := range defaultCategories {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO categories (user_id, name, kind, created_at)
			VALUES (?, ?, ?, ?)
		`, userID, cat.name, cat.kind, now)
		if err != nil {
			return nil, fmt.Errorf("insert default category %s: %w", cat.name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &User{
		ID:        userID,
		Email:     email,
		Timezone:  timezone,
		CreatedAt: now,
	}, nil
}

func (r *Repo) GetUserByEmail(ctx context.Context, email string) (*UserWithHash, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, timezone, created_at
		FROM users
		WHERE email = ?
	`, email)

	var u UserWithHash
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Timezone, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}

	return &u, nil
}

func (r *Repo) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, timezone, created_at
		FROM users
		WHERE id = ?
	`, userID)

	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.Timezone, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}

	return &u, nil
}

func (r *Repo) CreateSession(ctx context.Context, tokenHash string, userID int64, expiresAt string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, tokenHash, userID, expiresAt, now)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *Repo) GetSessionUser(ctx context.Context, tokenHash string) (*User, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	row := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.timezone, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?
	`, tokenHash, now)

	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.Timezone, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan session user: %w", err)
	}

	return &u, nil
}

func (r *Repo) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM sessions WHERE token_hash = ?
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
