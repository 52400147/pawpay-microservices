# Pet Profile Service

Service quản lý hồ sơ thú cưng, thuộc hệ thống PawPay. Viết bằng **Python (FastAPI)**.

## Chức năng
- REST API để Gateway và các service khác tra cứu/quản lý hồ sơ thú cưng.
- Trang Admin web để nhập liệu, sửa, xóa hồ sơ thú cưng trực tiếp.

## Model: PetProfile

| Trường | Kiểu | Ghi chú |
|---|---|---|
| id | int | Khóa chính, tự tăng |
| name | string | Tên thú cưng |
| species | string (optional) | Loài: chó, mèo, ... |
| user_id | int | ID chủ nuôi (tham chiếu sang User Service) |
| created_at | datetime | Tự sinh khi tạo |

## Cài đặt & chạy

```bash
cd pet-service
python -m venv venv
source venv/bin/activate      # Windows: venv\Scripts\activate
pip install -r requirements.txt
uvicorn main:app --reload --port 3002
```

Sau khi chạy:
- Trang Admin: http://localhost:3002/admin/pets
- API docs tự sinh (Swagger): http://localhost:3002/docs

## REST API

| Method | URI | Mô tả | Request Body | Response | Status |
|---|---|---|---|---|---|
| POST | `/api/pets` | Tạo hồ sơ mới | `{name, species, user_id}` | PetProfile | 201 |
| GET | `/api/pets` | Danh sách hồ sơ (lọc `?user_id=`) | — | `[PetProfile]` | 200 |
| GET | `/api/pets/{id}` | Tra cứu 1 hồ sơ theo id | — | PetProfile | 200 / 404 |
| PUT | `/api/pets/{id}` | Cập nhật hồ sơ | `{name?, species?}` | PetProfile | 200 / 404 |
| DELETE | `/api/pets/{id}` | Xóa hồ sơ | — | — | 204 / 404 |

Ví dụ response PetProfile:
```json
{
  "id": 1,
  "name": "Mochi",
  "species": "Mèo",
  "user_id": 5,
  "created_at": "2026-10-01T10:00:00"
}
```

## Dùng cho Use Case "Tra cứu hồ sơ"
Gateway/Payment Service gọi `GET /api/pets?user_id={id}` để lấy danh sách thú cưng của 1 chủ nuôi, hoặc `GET /api/pets/{id}` để lấy chi tiết 1 hồ sơ trước khi hiển thị các dịch vụ cần thanh toán.
