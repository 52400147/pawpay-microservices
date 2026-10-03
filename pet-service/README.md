# Pet Profile Service

Service quản lý hồ sơ thú cưng, thuộc hệ thống PawPay. Viết bằng **Python (FastAPI)**.
Đã cập nhật khớp với **hợp đồng API chung của nhóm** (file Apicontract.docx do Hương soạn).

## Model: PetProfile (bảng `pet_profiles`)

| Trường | Kiểu | Ghi chú |
|---|---|---|
| id | int | Khóa chính, tự tăng |
| name | string | Tên thú cưng |
| user_id | int | ID chủ nuôi (tham chiếu sang User Service) |

## Cài đặt & chạy

```bash
cd pet-service
python -m venv venv
source venv/bin/activate      # Windows: venv\Scripts\activate
pip install -r requirements.txt
uvicorn main:app --reload --port 3002
```

Sau khi chạy:
- API docs tự sinh (Swagger): http://localhost:3002/docs
- Trang Admin riêng (test nội bộ, không cần token): http://localhost:3002/admin/pets

## ⚠️ Bắt buộc trước khi ghép với cả nhóm

Service này tự kiểm tra JWT token, **không tin Gateway**. Mở file `auth.py`, đổi biến `JWT_SECRET`
thành đúng giá trị cả nhóm đã thống nhất (phải **giống hệt** bên User Service của Trâm và các
service khác), hoặc set qua biến môi trường:

```bash
export JWT_SECRET="gia_tri_ca_nhom_thong_nhat"
```

## REST API (theo đúng hợp đồng chung)

Mọi request đều cần header `Authorization: Bearer <token>`.

| Method | URI | Quyền | Mô tả | Response |
|---|---|---|---|---|
| GET | `/pets/{code}` | Đã đăng nhập | Tra cứu 1 hồ sơ theo mã (hiện dùng id) | 200 `{id, name, user_id}` / 404 |
| GET | `/pets` | Admin | Danh sách toàn bộ hồ sơ | 200 `[PetProfile]` |
| POST | `/pets` | Admin | Tạo hồ sơ mới | 201 PetProfile |
| PUT | `/pets/{id}` | Admin | Cập nhật hồ sơ | 200 PetProfile / 404 |
| DELETE | `/pets/{id}` | Admin | Xóa hồ sơ | 200 `{"message": ...}` / 404 |

Lỗi luôn trả dạng: `{"message": "nội dung lỗi tiếng Việt"}` kèm đúng HTTP status
(401 chưa đăng nhập, 403 không đủ quyền, 404 không tìm thấy).

## Test nhanh bằng curl

```bash
# Cần 1 token hợp lệ (role=admin) để test — xin Trâm cấp hoặc tự sinh token test tạm bằng cùng JWT_SECRET
curl -X POST http://localhost:3002/pets \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Mochi","user_id":2}'

curl http://localhost:3002/pets/1 -H "Authorization: Bearer <token>"
```

## Ghi chú
- Trang `/admin/pets` (giao diện web pastel) là công cụ riêng để tự test/nhập liệu nhanh,
  **không nằm trong hợp đồng API**, Gateway không gọi vào đây.
- Nếu sau này bảng `pet_profiles` cần thêm cột "mã hồ sơ" riêng (khác id), báo lại Hương để
  cập nhật hợp đồng vì `GET /pets/{code}` hiện đang dùng tạm `id` làm code.
