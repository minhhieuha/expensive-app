# SPEC: Web app quản lý chi tiêu (nhiều người dùng)

> Trạng thái: BẢN NHÁP, cần người dùng duyệt trước khi bắt đầu code. Chỉ `planner` được sửa file này.

## 1. Mục tiêu

Mỗi người dùng đăng ký tài khoản riêng, ghi lại các khoản chi/thu, đặt ngân sách theo tháng và xem tổng quan chi tiêu của mình.

## 2. Phạm vi MVP

### 2.1 Tài khoản
- Đăng ký (email, mật khẩu ≥ 8 ký tự), đăng nhập, đăng xuất.
- Mỗi user chỉ thấy dữ liệu của mình.
- Tiêu chí: email trùng bị từ chối; sai mật khẩu trả lỗi chung chung (không lộ email có tồn tại hay không); mật khẩu lưu dạng bcrypt.

### 2.2 Giao dịch
- Thêm/sửa/xóa giao dịch: số tiền (nguyên, > 0), loại (chi/thu), ngày, danh mục, ghi chú (tùy chọn, ≤ 500 ký tự).
- Danh sách có phân trang, lọc theo khoảng ngày, danh mục, loại; tìm theo ghi chú.
- Tiêu chí: giao dịch của user khác trả 404, không phải 403 (không lộ sự tồn tại).

### 2.3 Danh mục
- Có sẵn bộ danh mục mặc định khi đăng ký (Ăn uống, Di chuyển, Nhà ở, Giải trí, Sức khỏe, Khác...).
- User thêm/sửa/xóa danh mục riêng. Xóa danh mục đang có giao dịch: yêu cầu chuyển sang danh mục khác.

### 2.4 Ngân sách
- Đặt ngân sách theo tháng cho từng danh mục (số nguyên).
- Hiển thị đã chi / ngân sách; cảnh báo khi ≥ 80% và khi vượt 100%.

### 2.5 Dashboard
- Tổng chi, tổng thu, chênh lệch của tháng đang chọn.
- Biểu đồ chi theo danh mục; xu hướng chi 6 tháng gần nhất.

### 2.6 Xuất dữ liệu
- Xuất CSV giao dịch theo bộ lọc hiện tại (UTF-8 có BOM để Excel đọc được tiếng Việt).

## 3. Ngoài phạm vi (làm sau)

Đa tiền tệ, giao dịch định kỳ, đính kèm ảnh hóa đơn, chia sẻ giữa nhiều người, ứng dụng di động, quên mật khẩu qua email.

## 4. Yêu cầu phi chức năng

- Tiền là số nguyên ở mọi tầng.
- Cô lập dữ liệu theo user ở mọi endpoint (có test tự động).
- Giới hạn tốc độ đăng nhập (chống dò mật khẩu).
- Giao diện dùng được trên điện thoại (responsive).
- API trả lỗi nhất quán, không lộ stack trace.

## 5. Câu hỏi còn mở (planner hỏi người dùng)

- Đơn vị tiền mặc định: VND?
- Có cần ghi "thu" hay chỉ "chi"?
- Có cần ngôn ngữ giao diện nào ngoài tiếng Việt?
