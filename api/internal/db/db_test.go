package db

import (
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

	// User 2 inserting transaction referencing Category 10 (which belongs to user 1) MUST FAIL!
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

	// Invalid date format (slash instead of hyphen) should fail GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
	_, err = db.Exec(`
		INSERT INTO transactions (id, user_id, category_id, type, amount, occurred_on, created_at, updated_at)
		VALUES (2, 1, 1, 'expense', 1000, '2026/11/09', '2026-11-09T00:00:00Z', '2026-11-09T00:00:00Z')
	`)
	if err == nil {
		t.Error("expected check constraint violation for invalid occurred_on format, got nil")
	}
}
