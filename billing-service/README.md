# PawPay – Billing Service

**Ngôn ngữ:** Java 25 · Spring Boot 4.1.1 · JPA/H2/MySQL  
**Port:** `3003`  
**Nhánh:** `feature/billing-service`

---

## Chức năng

Quản lý hóa đơn dịch vụ thú cưng (spa, khám bệnh, ...) trong hệ thống PawPay.

---

## Model

| Trường | Kiểu | Ghi chú |
|---|---|---|
| `id` | Long | Auto-increment PK |
| `pet_profile_id` | Long | FK tới Pet Service |
| `service_type` | String | Loại dịch vụ (spa, khám bệnh...) |
| `amount` | BigDecimal | Số tiền, precision 15 scale 2 |
| `status` | Enum | `unpaid` hoặc `paid` |

---

## API Endpoints

| Method | URL | Auth | Body / Query | Response |
|---|---|---|---|---|
| `GET` | `/bills` | Admin token | `?pet_profile_id=&status=` | 200 array bills |
| `GET` | `/bills?pet_profile_id=1` | Any user | query | 200 array |
| `GET` | `/bills?status=unpaid` | Any user | query | 200 array |
| `GET` | `/bills/{id}` | Any user | — | 200 bill |
| `POST` | `/bills` | Admin | `{pet_profile_id, service_type, amount, status?}` | 201 bill |
| `PUT` | `/bills/{id}` | Admin | các trường cần sửa | 200 bill |
| `DELETE` | `/bills/{id}` | Admin | — | 200 `{message}` |
| `PUT` | `/bills/{id}/pay` | Any user | — | 200 bill / **409** nếu đã paid |

> **Note:** `PUT /bills/{id}/pay` là API nội bộ dành cho Payment Service.  
> Chỉ thành công **một lần** – gọi lần hai khi đã `paid` sẽ trả `409`.

---

## Cấu trúc thư mục

```
src/main/java/com/pawpay/billing/
├── BillingServiceApplication.java
├── controller/
│   └── BillController.java
├── dto/
│   ├── BillCreateRequest.java
│   ├── BillUpdateRequest.java
│   └── MessageResponse.java
├── entity/
│   ├── Bill.java
│   └── BillStatus.java
├── exception/
│   ├── ApiException.java
│   └── GlobalExceptionHandler.java
├── repository/
│   └── BillRepository.java
├── security/
│   ├── AuthUser.java
│   └── JwtService.java
└── service/
    └── BillService.java
```

---

## Chạy local (Dev)

```bash
# Clone & vào thư mục
cd billing-service

# Chạy (dùng H2 in-memory mặc định)
./mvnw spring-boot:run

# H2 Console (dev only)
# http://localhost:3003/h2-console
# JDBC URL: jdbc:h2:mem:billingdb
```

## Biến môi trường

| Biến | Mặc định | Ý nghĩa |
|---|---|---|
| `JWT_SECRET` | `dev-secret-doi-khi-ghep-nhom` | Khoá JWT chung cả nhóm |
| `DB_HOST` | `localhost` | MySQL host (production) |
| `DB_NAME` | `pawpay_billing` | MySQL database name |
| `DB_USER` | `root` | MySQL user |
| `DB_PASS` | _(rỗng)_ | MySQL password |

---

## Error Format

Mọi lỗi đều trả về:
```json
{ "message": "Mô tả lỗi" }
```

## JWT

Token phải có header `Authorization: Bearer <token>`.  
Payload cần có `user_id`, `role` (và tùy chọn `exp`).
