# Expense App: quy tắc cho mọi agent

Web app quản lý chi tiêu, nhiều người dùng. Dự án nhỏ, ưu tiên đơn giản và đúng.
Đọc thêm: `SPEC.md` (yêu cầu), `ARCHITECTURE.md` (thiết kế, schema, API), `TASKS.md` (việc cần làm).

## Stack (đã chốt, chi tiết trong ARCHITECTURE.md)

- **Frontend**: Angular (standalone) + TypeScript + SCSS, thư mục `web/`
- **Backend**: Go 1.22+, `net/http` ServeMux, thư mục `api/`
- **DB**: SQLite (`modernc.org/sqlite`), migration `.sql` đánh số trong `api/migrations/`
- **Auth**: email + mật khẩu (bcrypt), phiên lưu trong bảng `sessions`, cookie HttpOnly
- **Test**: `go test`, Jest/Karma, Playwright (`e2e/`)

## Lệnh chuẩn (chạy trước khi báo xong)

```
cd api && go build ./... && go vet ./... && go test ./...
cd web && npm run lint && npm test -- --watch=false && npm run build
cd e2e && npx playwright test
```

## Quy tắc bắt buộc

1. **Tiền là số nguyên** (đơn vị nhỏ nhất). Không float ở Go, SQL, TypeScript.
2. **Mọi truy vấn lọc theo `user_id`** lấy từ phiên đăng nhập, không bao giờ từ input của client. Hàm repository luôn nhận `userID` đầu tiên. Tài nguyên của người khác trả **404**.
3. Không hardcode secret. Cấu hình qua biến môi trường, có `.env.example`.
4. SQL luôn dùng tham số (`?`), không nối chuỗi.
5. Mỗi thay đổi hành vi phải có test đi kèm.
6. Làm trên branch `task/<id>-<tên-ngắn>`. **Không push hay merge vào `main`.** Merge do con người quyết định.
7. Mỗi task là một commit logic, message dạng `feat|fix|test|chore: mô tả`.
8. Ngày giao dịch là ngày lịch `YYYY-MM-DD` (không giờ). `created_at`/`updated_at` lưu UTC.
9. Thư viện được phép nằm ở ARCHITECTURE.md mục 2. Muốn thêm: dừng lại, báo cho người dùng.
10. Chỉ `planner` sửa `SPEC.md`, `ARCHITECTURE.md`. Chỉ agent chính (điều phối) sửa trạng thái trong `TASKS.md`; subagent báo kết quả về cho agent chính, không tự sửa file này.

## Quy ước code

- Go: lỗi trả về rõ ràng, không `panic` trong luồng request; handler mỏng, logic ở service, SQL ở repo; test dạng table-driven.
- Angular: standalone components, service gọi API, không logic nghiệp vụ trong template, không dùng `any`.
- API lỗi theo dạng `{ "error": { "code": "...", "message": "..." } }`.

## Điều cấm

- Đọc, sửa file `.env`, khóa, hoặc dữ liệu thật.
- Xóa hay làm yếu test để cho xanh. Test sai thì sửa test và giải thích.
- Tắt lint rule để qua lỗi.
- Chạy lệnh phá hủy (`rm -rf`, `git push --force`, `git reset --hard`) ngoài thư mục dự án hoặc trên `main`.

## Quy trình điều phối (dành cho agent chính)

Khi người dùng yêu cầu thực hiện một task (ví dụ "làm T03"), agent chính **không tự code** mà điều phối các subagent `developer`, `tester`, `reviewer` (mỗi subagent bắt đầu với context trống, nên prompt giao việc phải nêu rõ task ID và bảo chúng đọc `AGENTS.md`, `SPEC.md`, `ARCHITECTURE.md`, `TASKS.md`):

1. Đọc `TASKS.md`, xác nhận các task phụ thuộc đã `done`. Tạo branch `task/<id>-<tên-ngắn>`. Đặt trạng thái `doing`.
2. Gọi `developer`. Nhận báo cáo.
3. Gọi `tester` kèm báo cáo của developer. Có lỗi thì chuyển danh sách lỗi cho `developer` sửa rồi test lại (**tối đa 3 vòng**; quá thì dừng và báo người dùng).
4. Gọi `reviewer` (context mới). Nếu "Cần sửa" thì `developer` → `tester` → `reviewer` lại (**tối đa 2 vòng**).
5. Ghi kết quả vào `TASKS.md` (trạng thái + một dòng Nhật ký). Tóm tắt cho người dùng: đã làm gì, kết quả test thật, ý kiến reviewer. **Dừng ở đây để người dùng xem diff và merge.**
6. Chạy song song chỉ khi các task độc lập (không đụng cùng file, không phụ thuộc nhau): gọi mỗi vòng developer/tester/reviewer ở chế độ workspace `branch` (git worktree riêng). Chỉ agent chính được sửa `TASKS.md`.

Không bao giờ báo "xong" nếu chưa thấy kết quả chạy lệnh chuẩn.
