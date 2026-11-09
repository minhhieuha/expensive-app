# TASKS

Trạng thái: `todo` → `doing` → `testing` → `review` → `done`.
Mỗi task: một branch `task/<id>-<tên>`, qua tester và reviewer trước khi người dùng merge.
Chỉ agent chính (điều phối) và planner sửa file này. Chi tiết schema và API: xem `ARCHITECTURE.md`.

| ID | Task | Tiêu chí hoàn thành | Phụ thuộc | Trạng thái |
|----|------|---------------------|-----------|-----------|
| T01 | Khung dự án + CI | `api/` và `web/` build được; `go test`, `npm test`, lint chạy xanh; có `.env.example`; CI chạy 3 lệnh chuẩn; `GET /api/health` trả 200 | | review |
| T02 | DB + migration | Mở DB với pragma đúng (mục 4); runner migration; `0001_init.sql` đúng ARCHITECTURE.md; test: migration chạy được trên DB rỗng, khóa ngoại ghép từ chối giao dịch trỏ danh mục của user khác | T01 | todo |
| T03 | Auth backend | `register`, `login`, `logout`, `GET /me` theo mục 6; bcrypt; mật khẩu 8-72 byte; phiên trong bảng `sessions`; rate limit; seed danh mục mặc định khi đăng ký; test: hợp lệ, email trùng (409), sai mật khẩu (cùng thông báo với email không tồn tại), chưa đăng nhập (401), phiên hết hạn, sau logout token cũ không dùng được | T02 | todo |
| T04 | Middleware nền + cô lập dữ liệu | Auth middleware, CSRF header, security headers, lỗi chuẩn, helper `httpx.UserID`; bộ test mẫu "user A vs user B" dùng lại được cho các task sau | T03 | todo |
| T05 | API danh mục | CRUD `/api/categories`; xóa kèm `replace_with` trong một transaction; test cô lập user, chặn `replace_with` khác kind | T04 | todo |
| T06 | API giao dịch | CRUD + lọc + tìm + phân trang; validate amount, ngày, category cùng user và cùng kind; test hợp lệ, 4 case lỗi, cô lập user, `q` chứa `%` | T05 | todo |
| T07 | Frontend: khung + auth | Routing lazy, trang đăng ký/đăng nhập, guard, interceptor (X-Requested-With, 401 → login); test guard, interceptor, form | T04 | todo |
| T08 | Frontend: giao dịch + danh mục | Danh sách, bộ lọc, form thêm/sửa/xóa, quản lý danh mục; `money.pipe` VND; đủ trạng thái tải/lỗi/rỗng | T06, T07 | todo |
| T09 | Ngân sách (API + UI) | `GET/PUT/DELETE /api/budgets`; `status` tính bằng số nguyên (80% / vượt); UI đặt ngân sách, thanh tiến độ, cảnh báo | T06, T07 | todo |
| T10 | Dashboard (API + UI) | `/reports/summary`, `/by-category`, `/trend`; gom theo tháng bằng `occurred_on`; biểu đồ; test biên ngày 28-31 và 01, tháng không có dữ liệu | T06, T07 | todo |
| T11 | Xuất CSV | `GET /transactions/export` theo bộ lọc; UTF-8 BOM; test tiếng Việt, dấu phẩy, ngoặc kép, chống CSV injection (`=`, `+`, `-`, `@`) | T06 | todo |
| T12 | E2E + hoàn thiện | Playwright: đăng ký → thêm giao dịch → dashboard; hai user không thấy dữ liệu nhau; Dockerfile; README (cách chạy, biến môi trường) | T08-T11 | todo |

Gợi ý chạy song song (độc lập nhau): sau T04, làm T05 và T07 cùng lúc. Sau T06 và T07, làm T08, T09, T10 cùng lúc (mỗi task một worktree).

## Nhật ký

- 2026-11-09: T01 hoàn thành các vòng Developer -> Tester -> Reviewer trên branch `task/T01-khung-du-an`. Bộ 3 lệnh chuẩn (Go build/vet/test, Angular lint/test/build, Playwright e2e) đều đạt kết quả xanh. Chờ người dùng duyệt diff và merge vào `main`.
