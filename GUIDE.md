# Hướng dẫn chạy từng bước với Google Antigravity

Mỗi bước có phần **Kiểm tra** để biết đã đúng chưa. Chưa đúng thì đừng sang bước sau.

## Bước 0. Chuẩn bị

- Cài: Go 1.22+, Node.js LTS (kèm npm), Git.
- Tải Antigravity tại https://antigravity.google/download (macOS, Linux, Windows), đăng nhập bằng tài khoản Google. Hạn mức dùng agent gắn với gói Google AI của tài khoản.
- Antigravity có hai phần: **Antigravity IDE** (có trình soạn code) và **Antigravity 2.0** (trung tâm điều khiển agent độc lập). Dùng cái nào cũng được; hướng dẫn này dùng khung chat với agent trong một Project.
- Cần IDE **từ bản 1.20.5 trở lên** thì mới tự đọc `AGENTS.md`. Hãy cập nhật nếu đang dùng bản cũ.

## Bước 1. Đưa bộ file vào repo

```bash
unzip expense-app-agents.zip
cd expense-app
git init
git add .
git commit -m "chore: agent config and architecture"
git branch -M main
```

Nên đẩy lên một repo GitHub riêng tư để có bản sao lưu.

## Bước 2. Mở thành Project trong Antigravity

1. Mở Antigravity → thêm thư mục `expense-app` làm Project.
2. Vào cài đặt của Project, chỉnh hai mục (tên có thể hơi khác tùy phiên bản):
   - **Security Preset**: để mức yêu cầu bạn duyệt lệnh terminal và truy cập file, ít nhất trong những task đầu.
   - **Agent Behaviour**: bật yêu cầu bạn duyệt các artifact như kế hoạch triển khai (implementation plan).
3. Giữ tắt tùy chọn cho agent truy cập file ngoài dự án (mặc định đã tắt).

## Bước 3. Kiểm tra rule đã nạp

Hỏi agent mặc định: *"Tóm tắt các quy tắc bắt buộc của dự án này."*

**Kiểm tra:** câu trả lời phải nhắc tiền là số nguyên, lọc theo `user_id`, không merge vào `main`. Nếu agent không biết, kiểm tra: file `AGENTS.md` nằm ở **thư mục gốc** của Project; IDE đủ bản 1.20.5.

## Bước 4. Kiểm tra agent đã được nhận diện

Bốn file nằm ở `.agents/agents/`: `planner`, `developer`, `tester`, `reviewer`. Antigravity tự tìm file ở đó.

- `planner` là agent chính: chọn từ danh sách agent trong giao diện, hoặc chạy CLI: `agy --agent planner`.
- `developer`, `tester`, `reviewer` là subagent: agent chính gọi, bạn không chọn trực tiếp.
- Mở bảng `/agents` để xem agent nào đang chạy.

**Kiểm tra:** thấy `planner` trong danh sách. Nếu có subagent bị treo ngay khi gọi, nguyên nhân thường là tên công cụ sai trong mục `tools` (xem phần Xử lý sự cố).

## Bước 5. Chốt yêu cầu với planner

Chọn agent `planner`, gửi:

> Đọc SPEC.md. Hãy hỏi tôi 3 câu hỏi còn mở ở cuối file (đơn vị tiền, có ghi "thu" không, ngôn ngữ giao diện), rồi cập nhật SPEC.md, ARCHITECTURE.md và TASKS.md cho khớp câu trả lời. Chưa viết code.

Đọc kỹ `SPEC.md`, `ARCHITECTURE.md`, `TASKS.md` do planner cập nhật. Đồng ý thì commit:

```bash
git add . && git commit -m "docs: confirm spec and tasks"
```

**Kiểm tra:** mục "Câu hỏi còn mở" trong `SPEC.md` đã trống hoặc có trả lời.

## Bước 6. Chạy task đầu tiên (T01)

Quay lại agent mặc định (agent chính), gửi:

> Thực hiện task T01 theo mục "Quy trình điều phối" trong AGENTS.md.

Agent chính sẽ lần lượt gọi `developer` → `tester` → `reviewer`. Trong lúc chạy:

- Đọc **artifact kế hoạch** (implementation plan) nếu agent tạo ra, và nhận xét ngay nếu sai hướng.
- Duyệt từng lệnh terminal ở mức Security Preset bạn chọn. Từ chối lệnh lạ (xóa file hàng loạt, đụng `.env`, `git push`).
- Agent dừng khi `reviewer` kết luận. Báo cáo cuối cho bạn gồm kết quả test thật và ý kiến reviewer.

## Bước 7. Tự kiểm tra rồi merge

Đừng tin báo cáo; tự chạy lại:

```bash
git diff main...task/T01-khung-du-an --stat
git checkout task/T01-khung-du-an
cd api && go build ./... && go vet ./... && go test ./... && cd ..
cd web && npm run lint && npm test -- --watch=false && npm run build && cd ..
```

Đạt thì merge:

```bash
git checkout main && git merge --no-ff task/T01-khung-du-an
```

Bảo agent chính cập nhật `TASKS.md` (T01 → `done`) rồi commit.

## Bước 8. Lặp lại cho các task tiếp theo

Gửi: *"Thực hiện task T02 theo mục Quy trình điều phối trong AGENTS.md."* và làm lại bước 7.

**Chạy song song (khi đã quen):** các task độc lập có thể chạy cùng lúc, ví dụ sau T04 làm T05 và T07; sau T06 và T07 làm T08, T09, T10. Gửi:

> Chạy song song T05 và T07, mỗi task trong một workspace `branch` riêng (git worktree), mỗi task đủ vòng developer → tester → reviewer. Chỉ bạn được sửa TASKS.md.

Sau đó merge từng branch một. Gặp xung đột thì xử lý trước khi merge branch kế tiếp.

## Bước 9. Quy tắc giữ cho dự án khỏe

- **Một task mỗi lần** cho tới khi bạn tin vào bộ agent. Song song sau.
- **Luôn tự xem diff** trước khi merge, đặc biệt các file auth, migration và mọi hàm repository.
- Khi agent làm sai điều gì lặp lại: sửa **prompt agent** hoặc **`AGENTS.md`**, đừng chỉ sửa code.
- Task bị kẹt quá 3 vòng: dừng, đọc lỗi, làm rõ tiêu chí trong `TASKS.md` rồi chạy lại.
- Commit sau mỗi task để có điểm quay lại.

## Xử lý sự cố

| Triệu chứng | Nguyên nhân thường gặp | Cách xử lý |
|---|---|---|
| Agent không biết quy tắc dự án | `AGENTS.md` không ở thư mục gốc, hoặc IDE < 1.20.5 | Đưa file về gốc, cập nhật IDE |
| Subagent treo, không phản hồi | Tên công cụ trong `tools` sai chính tả hoặc không tồn tại | Chỉ dùng tên đã xác thực: `view_file`, `grep_search`, `replace_file_content`, `run_command`; kiểm tra lại từng file agent |
| Agent không tạo được file mới | Bộ `tools` thiếu công cụ tạo file | Agent dùng `run_command` để tạo; hoặc thêm đúng tên công cụ tạo file mà IDE của bạn hiển thị vào `tools` |
| Agent chính tự code thay vì gọi subagent | Prompt không nhắc quy trình | Nói rõ: "theo mục Quy trình điều phối trong AGENTS.md, đừng tự code" |
| Rule trong `.agents/rules/` không có tác dụng | Thiếu frontmatter `trigger` (file bị bỏ qua im lặng) | Dùng `AGENTS.md` ở gốc, hoặc thêm `trigger` hợp lệ |
| Lệnh bị chặn hoặc hỏi duyệt liên tục | Security Preset quá chặt | Nới dần theo từng loại lệnh, đừng bật tự chạy mọi thứ |
| Hết hạn mức | Giới hạn gói Google AI | Chờ làm mới, hoặc đổi `model` của subagent nhẹ như tester sang `flash` |

## Giới hạn của bộ cấu hình này

- Mình chỉ xác thực được bốn tên công cụ ở trên từ tài liệu chính thức; tài liệu cho biết đó là ví dụ, không phải danh sách đầy đủ. Hãy thử trên T01 và chỉnh `tools` nếu IDE báo khác.
- Việc agent chính tự gọi subagent theo `AGENTS.md` dựa vào việc agent làm theo hướng dẫn, chưa được kiểm thử trên máy bạn.
