package db

import (
	"strings"
	"testing"
)

func TestOpenAndPragmas(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fkEnabled); err != nil {
		t.Fatalf("failed to query pragma foreign_keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("expected foreign_keys=1, got %d", fkEnabled)
	}

	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		t.Fatalf("failed to query pragma busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("expected busy_timeout=5000, got %d", busyTimeout)
	}
}

func TestMigrateCleanDB(t *testing.T) {
	db := OpenTestDB(t)

	tables := []string{"users", "sessions", "categories", "transactions", "budgets", "schema_migrations"}
	for _, table := range tables {
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&exists)
		if err != nil {
			t.Fatalf("failed to check table %s: %v", table, err)
		}
		if exists != 1 {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db := OpenTestDB(t)

	// Running Migrate a second time should succeed without error
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate call failed: %v", err)
	}
}

func TestCompositeForeignKeyEnforcement(t *testing.T) {
	db := OpenTestDB(t)

	// Create User 1 and User 2
	_, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES 
			(1, 'user1@example.com', 'hash1', '2026-11-09T00:00:00Z'),
			(2, 'user2@example.com', 'hash2', '2026-11-09T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("failed to insert users: %v", err)
	}

	// Create Category 1 for User 1
	_, err = db.Exec(`
		INSERT INTO categories (id, user_id, name, kind, created_at)
		VALUES (10, 1, 'Ăn uống', 'expense', '2026-11-09T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("failed to insert category for user 1: %v", err)
	}

	// User 1 inserting transaction with their own category (10) must succeed
	_, err = db.Exec(`
		INSERT INTO transactions (id, user_id, category_id, type, amount, occurred_on, note, created_at, updated_at)
		VALUES (100, 1, 10, 'expense', 50000, '2026-11-09', 'Cơm trưa', '2026-11-09T00:00:00Z', '2026-11-09T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("user 1 inserting transaction failed: %v", err)
	}

	// User 2 inserting transaction referencing Category 10 (belonging to user 1) MUST FAIL!
	_, err = db.Exec(`
		INSERT INTO transactions (id, user_id, category_id, type, amount, occurred_on, note, created_at, updated_at)
		VALUES (101, 2, 10, 'expense', 60000, '2026-11-09', 'Lậu', '2026-11-09T00:00:00Z', '2026-11-09T00:00:00Z');
	`)
	if err == nil {
		t.Fatal("expected composite foreign key constraint violation when user 2 references user 1's category, but got nil error")
	}

	// User 2 inserting budget referencing Category 10 (belonging to user 1) MUST ALSO FAIL!
	_, err = db.Exec(`
		INSERT INTO budgets (id, user_id, category_id, month, amount)
		VALUES (200, 2, 10, '2026-11', 1000000);
	`)
	if err == nil {
		t.Fatal("expected composite foreign key constraint violation when user 2 sets budget for user 1's category, but got nil error")
	}
}

func TestCategoryDeleteRestrict(t *testing.T) {
	db := OpenTestDB(t)

	_, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES (1, 'u@example.com', 'h', '2026-11-09T00:00:00Z');
		INSERT INTO categories (id, user_id, name, kind, created_at)
		VALUES (1, 1, 'Ăn uống', 'expense', '2026-11-09T00:00:00Z');
		INSERT INTO transactions (id, user_id, category_id, type, amount, occurred_on, created_at, updated_at)
		VALUES (1, 1, 1, 'expense', 1000, '2026-11-09', '2026-11-09T00:00:00Z', '2026-11-09T00:00:00Z');
	`)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Deleting category 1 while transaction 1 still references it must fail due to ON DELETE RESTRICT
	_, err = db.Exec("DELETE FROM categories WHERE id = 1")
	if err == nil {
		t.Fatal("expected error deleting category with existing transactions (ON DELETE RESTRICT), got nil")
	}
}

func TestUserCascadeDelete(t *testing.T) {
	db := OpenTestDB(t)

	_, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES (1, 'u@example.com', 'h', '2026-11-09T00:00:00Z');
		INSERT INTO sessions (token_hash, user_id, expires_at, created_at)
		VALUES ('tok1', 1, '2026-12-09T00:00:00Z', '2026-11-09T00:00:00Z');
		INSERT INTO categories (id, user_id, name, kind, created_at)
		VALUES (1, 1, 'Ăn uống', 'expense', '2026-11-09T00:00:00Z');
		INSERT INTO budgets (id, user_id, category_id, month, amount)
		VALUES (1, 1, 1, '2026-11', 500000);
	`)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Deleting user 1 should cascade delete sessions, categories, budgets
	_, err = db.Exec("DELETE FROM users WHERE id = 1")
	if err != nil {
		t.Fatalf("failed to delete user 1: %v", err)
	}

	var sessionCount, catCount, budgetCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = 1").Scan(&sessionCount)
	_ = db.QueryRow("SELECT COUNT(*) FROM categories WHERE user_id = 1").Scan(&catCount)
	_ = db.QueryRow("SELECT COUNT(*) FROM budgets WHERE user_id = 1").Scan(&budgetCount)

	if sessionCount != 0 || catCount != 0 || budgetCount != 0 {
		t.Errorf("expected 0 dependent records after user cascade delete, got sessions=%d, categories=%d, budgets=%d",
			sessionCount, catCount, budgetCount)
	}
}

func TestEmailUniqueCaseInsensitive(t *testing.T) {
	db := OpenTestDB(t)

	_, err := db.Exec(`INSERT INTO users (email, password_hash, created_at) VALUES ('test@example.com', 'h1', '2026-11-09T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert user failed: %v", err)
	}

	// Case-insensitive duplicate email must fail
	_, err = db.Exec(`INSERT INTO users (email, password_hash, created_at) VALUES ('Test@EXAMPLE.COM', 'h2', '2026-11-09T00:00:00Z')`)
	if err == nil {
		t.Error("expected UNIQUE COLLATE NOCASE violation for duplicate email with different casing, got nil")
	}
}

func TestConstraints(t *testing.T) {
	db := OpenTestDB(t)

	_, err := db.Exec(`INSERT INTO users (id, email, password_hash, created_at) VALUES (1, 'u@example.com', 'h', '2026-11-09T00:00:00Z')`)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	_, err = db.Exec(`INSERT INTO categories (id, user_id, name, kind, created_at) VALUES (1, 1, 'Ăn uống', 'expense', '2026-11-09T00:00:00Z')`)
	if err != nil {
		t.Fatalf("failed to insert category: %v", err)
	}

	// Amount <= 0 should fail
	_, err = db.Exec(`
		INSERT INTO transactions (id, user_id, category_id, type, amount, occurred_on, created_at, updated_at)
		VALUES (1, 1, 1, 'expense', 0, '2026-11-09', '2026-11-09T00:00:00Z', '2026-11-09T00:00:00Z')
	`)
	if err == nil {
		t.Error("expected check constraint violation for amount=0, got nil")
	}

	// Amount > 1e12 should fail
	_, err = db.Exec(`
		INSERT INTO transactions (id, user_id, category_id, type, amount, occurred_on, created_at, updated_at)
		VALUES (2, 1, 1, 'expense', 1000000000001, '2026-11-09', '2026-11-09T00:00:00Z', '2026-11-09T00:00:00Z')
	`)
	if err == nil {
		t.Error("expected check constraint violation for amount > 1e12, got nil")
	}

	// Invalid date format (slash instead of hyphen) should fail GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
	_, err = db.Exec(`
		INSERT INTO transactions (id, user_id, category_id, type, amount, occurred_on, created_at, updated_at)
		VALUES (3, 1, 1, 'expense', 1000, '2026/11/09', '2026-11-09T00:00:00Z', '2026-11-09T00:00:00Z')
	`)
	if err == nil {
		t.Error("expected check constraint violation for invalid occurred_on format, got nil")
	}

	// Note length > 500 should fail
	longNote := strings.Repeat("a", 501)
	_, err = db.Exec(`
		INSERT INTO transactions (id, user_id, category_id, type, amount, occurred_on, note, created_at, updated_at)
		VALUES (4, 1, 1, 'expense', 1000, '2026-11-09', ?, '2026-11-09T00:00:00Z', '2026-11-09T00:00:00Z')
	`, longNote)
	if err == nil {
		t.Error("expected check constraint violation for note length > 500, got nil")
	}
}
