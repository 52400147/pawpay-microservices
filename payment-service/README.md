# PawPay – Payment/Transaction Service (Go, cổng 3004)

## Chạy thử
```bash
# 1. PostgreSQL: tạo database
createdb pawpay_payment            # bảng tự tạo từ schema.sql khi service khởi động

# 2. Chạy service (JWT_SECRET phải đồng nhất)
export JWT_SECRET=...              
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/pawpay_payment?sslmode=disable"
export USER_SERVICE_URL=http://localhost:3001
export BILLING_SERVICE_URL=http://localhost:3003
export OTP_SERVICE_URL=http://localhost:3005
go run .
```

## API (đúng Apicontract.docx)
| API | Quyền | Kết quả |
| --- | --- | --- |
| POST /payments `{bill_id, pet_profile_id}` | user | 201 `{transaction_id}`; 404 / 409 / 422 |
| POST /payments/{id}/confirm `{otp}` | chủ giao dịch | 200; 400 OTP sai; 409 hóa đơn đã trả; 422 thiếu số dư |
| GET /transactions | user | mảng giao dịch của người đăng nhập |
| GET /admin/transactions[?status=&user_id=] | admin | mảng toàn bộ giao dịch |

## API nội bộ thống nhất 
Payment gọi các API này bằng token nội bộ (JWT cùng JWT_SECRET, role=admin, hết hạn 1 phút).

| Gọi tới | API | Yêu cầu từ phía service kia |
| --- | --- | --- |
| User (Trâm) | `POST /users/{id}/deduct {amount}` | Trừ **atomic** (`UPDATE ... SET balance=balance-$1 WHERE id=$2 AND balance>=$1`). Thiếu số dư: 422 |
| User (Trâm) | `POST /users/{id}/refund {amount}` | Cộng lại số dư (dùng khi bù trừ) |
| User (Trâm) | `GET /users/me` | Trả `id, email, balance` (đã có trong hợp đồng) |
| Billing (Sang) | `GET /bills?pet_profile_id=` và `GET /bills` | Trả mảng hóa đơn; token admin được xem tất cả |
| Billing (Sang) | `PUT /bills/{id}/pay` | Chỉ thành công 1 lần, lần 2 trả 409 |
| OTP (Thịnh) | `POST /otp/generate {transaction_id, email}` | 2xx khi đã gửi |
| OTP (Thịnh) | `POST /otp/verify {transaction_id, code}` | 200 nếu đúng; sai/hết hạn: 400 `{message}` |
| OTP (Thịnh) | `POST /notifications/success {email, transaction_id, bill_id, amount}` | Gửi email xác nhận |

## Xử lý đồng thời (yêu cầu 6)
Trong `confirm`, mọi bước chạy trong một DB transaction:
1. `SELECT ... FOR UPDATE` dòng giao dịch → hai lần confirm cùng một giao dịch chỉ một lần thắng.
2. `pg_advisory_xact_lock` theo `bill_id` → hai người trả cùng một hóa đơn bị xếp hàng.
3. `pg_advisory_xact_lock` theo `user_id` → cùng một user trả nhiều hóa đơn song song không trừ quá số dư.
4. Chốt chặn cuối: unique index `uq_transactions_bill_success` (mỗi hóa đơn tối đa một giao dịch success).

Thứ tự khóa cố định (giao dịch → hóa đơn → user) nên không deadlock. Số dư và hóa đơn nằm ở DB của service
khác nên không thể `FOR UPDATE` trực tiếp; vì vậy dùng khóa logic ở Payment, và User/Billing vẫn phải tự atomic.
Nếu đã trừ tiền mà đánh dấu hóa đơn lỗi (409) thì Payment gọi `refund` để bù trừ.

## Đã kiểm thử (PostgreSQL thật + service giả)
Đúng luồng tạo → OTP sai (400) → confirm; 5 request confirm cùng lúc chỉ 1 thành công; 2 người trả cùng một hóa đơn
chỉ 1 thành công, số dư người kia không bị trừ; hóa đơn đã trả 409; thiếu số dư 422; thiếu token 401; user thường
gọi admin 403.
