# ARCHITECTURE: Expense App

> Chỉ `planner` được sửa file này. Mọi thay đổi kiến trúc phải ghi thêm một dòng vào mục 12 (Nhật ký quyết định).

## 1. Tổng quan

```
Trình duyệt (Angular SPA)
        │  HTTPS, JSON, cookie phiên (HttpOnly)
        ▼
┌───────────────────────────────────────────┐
│ Go server (một binary)                    │
│  ├─ /api/*  → middleware → handler        │
│  │             → service → repository     │
│  └─ /*      → file tĩnh của Angular       │
└───────────────────┬───────────────────────┘
                    ▼
            SQLite (1 file, WAL)
```

- Một binary Go phục vụ cả API lẫn file tĩnh Angular, nên cùng origin, không cần CORS.
- Khi dev: `ng serve` chạy riêng và proxy `/api` sang Go (`web/proxy.conf.json`), cookie vẫn cùng origin.

## 2. Quyết định kỹ thuật

| Hạng mục | Chọn | Lý do / phương án đã loại |
|---|---|---|
| Router | `net/http` ServeMux (Go 1.22+, hỗ trợ method + path param) | Đủ dùng, không thêm thư viện. Loại: chi, gin |
| DB | SQLite, driver `modernc.org/sqlite` | Không cần CGO, deploy dễ. Truy cập qua repository nên đổi Postgres được |
| Migration | File `.sql` đánh số, nhúng bằng `embed`, runner tự viết (~50 dòng) | Tránh thêm thư viện. Loại: goose, migrate |
| Mật khẩu | bcrypt (`golang.org/x/crypto/bcrypt`), cost 12 | Chuẩn, đơn giản |
| Phiên | Token ngẫu nhiên 32 byte, **lưu SHA-256 trong bảng `sessions`**, gửi qua cookie | Đăng xuất và thu hồi thật sự được. Loại: JWT (không thu hồi được, phức tạp hơn cần thiết) |
| Frontend | Angular standalone + signals + reactive forms | Không dùng NgRx: app nhỏ |
| Biểu đồ | `chart.js` | Nhẹ, đủ cho cột/tròn/đường |
| Test | `go test`, Jest hoặc Karma, Playwright | Xem mục 10 |

**Thư viện được phép:** `modernc.org/sqlite`, `golang.org/x/crypto`, `chart.js`, Playwright. Thêm thư viện khác phải ghi lý do ở mục 12.

## 3. Cấu trúc backend (`api/`)

```
api/
├── cmd/server/main.go          # đọc config, mở DB, chạy migration, dựng router
├── migrations/                 # 0001_init.sql, 0002_...
└── internal/
    ├── config/                 # đọc biến môi trường
    ├── db/                     # Open (pragma), Migrate, helper test
    ├── httpx/                  # JSON helper, lỗi chuẩn, middleware (auth, csrf, ratelimit, log, recover)
    ├── auth/                   # handler.go service.go repo.go
    ├── category/               # handler.go service.go repo.go
    ├── transaction/            # handler.go service.go repo.go csv.go
    ├── budget/                 # handler.go service.go repo.go
    └── report/                 # handler.go service.go repo.go
```

Mỗi package miền có 3 tầng, mỗi tầng một việc:

- **handler**: đọc request, validate hình thức, gọi service, ghi response. Không có SQL, không có luật nghiệp vụ.
- **service**: luật nghiệp vụ (ví dụ category.kind phải khớp transaction.type). Không biết HTTP.
- **repo**: SQL. **Mọi hàm nhận `userID` làm tham số đầu tiên và luôn có `WHERE user_id = ?`.** Không có hàm nào lấy dữ liệu chỉ bằng `id`.

## 4. Cơ sở dữ liệu

Pragma khi mở kết nối: `foreign_keys=ON`, `journal_mode=WAL`, `busy_timeout=5000`. Đặt `SetMaxOpenConns(1)` để tránh `SQLITE_BUSY` (đủ cho quy mô này).

```sql
-- migrations/0001_init.sql
CREATE TABLE users (
  id            INTEGER PRIMARY KEY,
  email         TEXT NOT NULL UNIQUE COLLATE NOCASE,
  password_hash TEXT NOT NULL,
  timezone      TEXT NOT NULL DEFAULT 'Asia/Ho_Chi_Minh',
  created_at    TEXT NOT NULL                      -- UTC, RFC3339
);

CREATE TABLE sessions (
  token_hash TEXT PRIMARY KEY,                     -- SHA-256 (hex) của token trong cookie
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
  UNIQUE (id, user_id)                             -- để bảng khác tham chiếu kèm user_id
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
```

Điểm đáng chú ý: khóa ngoại ghép `(category_id, user_id)` khiến **chính DB từ chối** giao dịch của user A trỏ tới danh mục của user B, kể cả khi code có lỗi.

Danh mục mặc định (seed khi đăng ký, trong cùng transaction với việc tạo user):
- Chi: Ăn uống, Di chuyển, Nhà ở, Hóa đơn, Giải trí, Sức khỏe, Học tập, Khác
- Thu: Lương, Thưởng, Khác

## 5. API

Tiền-tố `/api`. Chỉ nhận và trả JSON (`Content-Type: application/json`), trừ export CSV.

| Method + Path | Mô tả |
|---|---|
| `POST /auth/register` | `{email, password}` → tạo user, seed danh mục, đăng nhập luôn |
| `POST /auth/login` | `{email, password}` → set cookie |
| `POST /auth/logout` | xóa phiên |
| `GET /me` | thông tin user hiện tại |
| `GET /categories` | danh sách (lọc `?kind=`) |
| `POST /categories` | `{name, kind}` |
| `PATCH /categories/{id}` | đổi tên |
| `DELETE /categories/{id}?replace_with={id}` | nếu còn giao dịch thì bắt buộc `replace_with` (cùng kind), chuyển giao dịch rồi xóa, **trong một transaction DB** |
| `GET /transactions` | `?from&to&category_id&type&q&page&page_size` |
| `POST /transactions` | `{type, amount, occurred_on, category_id, note}` |
| `GET/PATCH/DELETE /transactions/{id}` | |
| `GET /transactions/export` | cùng bộ lọc, trả CSV |
| `GET /budgets?month=YYYY-MM` | mỗi dòng: `category_id, amount, spent, status` |
| `PUT /budgets` | upsert `{category_id, month, amount}` |
| `DELETE /budgets/{id}` | |
| `GET /reports/summary?month=` | `{total_expense, total_income, balance}` |
| `GET /reports/by-category?month=` | chi theo danh mục |
| `GET /reports/trend?month=&months=6` | chi/thu theo tháng, kết thúc ở `month` |

**Quy ước**
- Phân trang: `page` (từ 1), `page_size` (mặc định 20, tối đa 100). Trả `{items, page, page_size, total}`. Sắp xếp `occurred_on DESC, id DESC`.
- Tìm `q`: `LIKE` trên `note`, escape `%` và `_`.
- `status` của ngân sách tính bằng số nguyên: `ok`; `warning` khi `spent*100 >= amount*80`; `exceeded` khi `spent > amount`.
- Lỗi luôn có dạng `{"error": {"code": "...", "message": "...", "fields": {...}}}`:

| HTTP | code | Khi nào |
|---|---|---|
| 400 | `bad_request` | JSON hỏng, tham số sai kiểu |
| 401 | `unauthenticated` / `invalid_credentials` | chưa đăng nhập / sai email hoặc mật khẩu (cùng một thông báo) |
| 404 | `not_found` | không tồn tại **hoặc thuộc user khác** |
| 409 | `conflict` | email trùng, tên danh mục trùng |
| 422 | `validation_failed` | sai luật (amount ≤ 0, ngày sai...), kèm `fields` |
| 429 | `rate_limited` | đăng nhập sai quá nhiều |
| 500 | `internal` | không lộ chi tiết, chỉ log phía server |

Đăng ký trùng email trả 409, tức có lộ email tồn tại. Đây là đánh đổi chấp nhận cho MVP.

## 6. Xác thực và cô lập dữ liệu

**Đăng nhập**
1. Chuẩn hóa email (trim, lowercase). Mật khẩu 8 đến 72 byte (bcrypt cắt từ byte 72).
2. Không tìm thấy user vẫn chạy một lần so sánh bcrypt với hash giả để thời gian phản hồi không lộ email có tồn tại.
3. Đúng thì tạo token 32 byte từ `crypto/rand`, lưu SHA-256 vào `sessions` (hạn 30 ngày), gửi cookie `session=<token>; HttpOnly; Secure; SameSite=Lax; Path=/`.
4. `Secure` bật theo `COOKIE_SECURE` (tắt khi chạy `http://localhost`).

**Mỗi request `/api/*` (trừ register/login)**
`auth middleware` → băm cookie → tra `sessions` còn hạn → đặt `userID` vào `context`. Handler lấy `userID` từ context bằng một helper duy nhất (`httpx.UserID(ctx)`), **không bao giờ đọc `user_id` từ body, query hay path**.

**Chống CSRF**: cookie `SameSite=Lax` + yêu cầu `Content-Type: application/json` + mọi request ghi (`POST/PUT/PATCH/DELETE`) phải có header `X-Requested-With: expense-app` (Angular interceptor tự thêm). Thiếu header thì 403.

**Rate limit**: đăng nhập tối đa 5 lần/phút cho mỗi cặp (IP, email), giữ trong bộ nhớ (một instance là đủ).

**Header an toàn**: `Content-Security-Policy` (chỉ `self`), `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin`.

**Ba lớp bảo vệ dữ liệu theo user** (mỗi lớp có test riêng):
1. Repository luôn lọc `user_id`.
2. Trả 404 cho tài nguyên của người khác.
3. Khóa ngoại ghép ở DB (mục 4).

## 7. Tiền và ngày

- **Tiền**: số nguyên đơn vị nhỏ nhất (VND: đồng, không có phần lẻ). Cột `INTEGER`, Go `int64`, TypeScript `number` (an toàn tới 2^53, và đã chặn trên `1e12`). Tuyệt đối không float. Phép chia phần trăm làm bằng số nguyên.
- **Ngày giao dịch** (`occurred_on`): ngày lịch `YYYY-MM-DD` do người dùng chọn, **không có giờ, không có múi giờ**. Nhờ vậy gom theo tháng là so khớp chuỗi (`occurred_on >= '2026-11-01' AND occurred_on < '2026-12-01'`), không có lỗi lệch múi giờ.
- `created_at`, `updated_at`, `expires_at`: UTC, RFC3339.
- `users.timezone` chỉ dùng để biết "hôm nay" và "tháng hiện tại" khi UI chọn mặc định.
- **CSV**: `encoding/csv`, UTF-8 có BOM, cột `date,type,category,amount,note`. Ô bắt đầu bằng `=`, `+`, `-`, `@` thì thêm dấu `'` phía trước (chống CSV injection).

## 8. Frontend (`web/`)

```
web/src/app/
├── core/                 # auth.service, api client, interceptors (cookie, X-Requested-With, 401→login), guard, models
├── shared/               # money.pipe (Intl vi-VN, VND), date helper, confirm dialog, empty-state
└── features/
    ├── auth/             # login, register
    ├── transactions/     # list + filter, form
    ├── categories/
    ├── budgets/
    └── dashboard/
```

- Routing lazy-load theo feature; `authGuard` chặn route cần đăng nhập.
- Trạng thái: signals trong service theo feature. Không NgRx.
- Form: reactive forms, validate khớp luật backend (backend vẫn là nguồn sự thật).
- Mọi component xử lý đủ ba trạng thái: đang tải, lỗi, rỗng.
- Không dùng `innerHTML` với dữ liệu người dùng; để Angular tự escape.
- Nhập tiền: người dùng gõ có dấu phân cách, chuyển thành số nguyên trước khi gửi.
- Responsive từ 360px.

## 9. Cấu hình (biến môi trường, `.env.example`)

| Biến | Mặc định | Ý nghĩa |
|---|---|---|
| `PORT` | `8080` | cổng HTTP |
| `DB_PATH` | `./data/expense.db` | đường dẫn file SQLite |
| `COOKIE_SECURE` | `true` | đặt `false` khi dev qua http |
| `SESSION_TTL_HOURS` | `720` | hạn phiên |
| `STATIC_DIR` | `../web/dist/web/browser` | thư mục file tĩnh (bỏ trống nếu chỉ chạy API) |

Không có secret nào cần cấu hình: token phiên sinh ngẫu nhiên và chỉ lưu bản băm.

## 10. Chiến lược kiểm thử

- **Backend**: table-driven. Mỗi test tạo DB SQLite trong bộ nhớ riêng và chạy **migration thật**. Test handler qua `httptest` với server dựng đầy đủ middleware.
- **Bộ test cô lập (bắt buộc cho mọi endpoint dữ liệu)**: tạo user A và B; A tạo dữ liệu; B thử đọc/sửa/xóa/liệt kê/xuất CSV → luôn 404 hoặc danh sách rỗng. Chưa đăng nhập → 401.
- **Frontend**: test service và component chính (form, guard, interceptor), mock HTTP.
- **E2E (Playwright)**: (1) đăng ký → thêm giao dịch → xem dashboard; (2) hai user không thấy dữ liệu của nhau.
- **Test biên cần có**: amount 0, âm, quá `1e12`; ngày 29/02, 31/12, 01/01; ghi chú 500 và 501 ký tự; tiếng Việt có dấu; `<script>` trong ghi chú; `page_size` lớn; `q` chứa `%`.

## 11. Build và triển khai

```
# Dev
cd api && go run ./cmd/server          # :8080
cd web && npm start                    # :4200, proxy /api → :8080

# Build
cd web && npm run build
cd api && go build -o ../bin/server ./cmd/server
```

Docker đa tầng: tầng 1 build Angular, tầng 2 build Go, tầng 3 (image nhỏ) chứa binary + `web/dist`. Gắn volume cho thư mục chứa `DB_PATH`. Sao lưu: dùng `sqlite3 expense.db ".backup backup.db"` (an toàn khi server đang chạy), không copy file thô.

## 12. Nhật ký quyết định

| Ngày | Quyết định | Lý do |
|---|---|---|
| 2026-11-09 | Phiên bằng bảng `sessions` thay vì JWT | Thu hồi được, ít thứ có thể sai |
| 2026-11-09 | `occurred_on` là ngày lịch, không phải timestamp | Loại bỏ lỗi múi giờ khi gom theo tháng |
| 2026-11-09 | Khóa ngoại ghép `(category_id, user_id)` | DB tự chặn dữ liệu chéo user |
| 2026-11-09 | ServeMux thay vì chi | Không thêm thư viện |
