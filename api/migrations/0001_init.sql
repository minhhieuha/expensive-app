CREATE TABLE users (
  id            INTEGER PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE COLLATE NOCASE,
  password_hash TEXT NOT NULL,
  timezone      TEXT NOT NULL DEFAULT 'Asia/Ho_Chi_Minh',
  created_at    TEXT NOT NULL
);

CREATE TABLE sessions (
  token_hash TEXT PRIMARY KEY,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_sessions_user ON sessions(user_id);

CREATE TABLE categories (
  id         INTEGER PRIMARY KEY,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 50),
  kind       TEXT NOT NULL CHECK (kind IN ('expense','income')),
  created_at TEXT NOT NULL,
  UNIQUE (user_id, kind, name),
  UNIQUE (id, user_id)
);

CREATE TABLE transactions (
  id          INTEGER PRIMARY KEY,
  user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  category_id INTEGER NOT NULL,
  type        TEXT NOT NULL CHECK (type IN ('expense','income')),
  amount      INTEGER NOT NULL CHECK (amount > 0 AND amount <= 1000000000000),
  occurred_on TEXT NOT NULL CHECK (occurred_on GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  note        TEXT NOT NULL DEFAULT '' CHECK (length(note) <= 500),
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL,
  FOREIGN KEY (category_id, user_id) REFERENCES categories(id, user_id) ON DELETE RESTRICT
);
CREATE INDEX idx_tx_user_date ON transactions(user_id, occurred_on DESC, id DESC);
CREATE INDEX idx_tx_user_cat  ON transactions(user_id, category_id);

CREATE TABLE budgets (
  id          INTEGER PRIMARY KEY,
  user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  category_id INTEGER NOT NULL,
  month       TEXT NOT NULL CHECK (month GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]'),
  amount      INTEGER NOT NULL CHECK (amount > 0),
  UNIQUE (user_id, category_id, month),
  FOREIGN KEY (category_id, user_id) REFERENCES categories(id, user_id) ON DELETE CASCADE
);
