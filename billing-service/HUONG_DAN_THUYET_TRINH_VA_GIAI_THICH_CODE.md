# HƯỚNG DẪN GIẢI THÍCH TOÀN BỘ CODE & KỊCH BẢN THUYẾT TRÌNH BÁO CÁO
**Dự án:** PawPay Microservices  
**Dịch vụ:** `billing-service` (Quản lý dịch vụ & Hóa đơn thanh toán)  
**Sinh viên phụ trách:** Minh Sang (MSSV: 52400154)  
**Công nghệ sử dụng:** Java 25, Spring Boot 4.1.1, Spring Data JPA, H2 Database / MySQL, HMAC-SHA256 JWT  

---

## PHẦN 1: GIẢI THÍCH CHI TIẾT TỪNG FILE TRONG SOURCE CODE

Cấu trúc dự án được thiết kế theo **Kiến trúc phân lớp chuẩn (Layered Architecture)** trong Spring Boot:

```
billing-service/
├── pom.xml                                      # Quản lý thư viện & build tool
├── src/main/resources/application.properties    # Cấu hình cổng, DB, JWT secret
├── docs/use-case-diagram.png                    # Sơ đồ Use Case quản lý hóa đơn
└── src/main/java/com/pawpay/billing/
    ├── BillingServiceApplication.java           # Entry point khởi chạy ứng dụng
    ├── entity/                                  # Tầng Entity (Ánh xạ bảng CSDL)
    │   ├── Bill.java
    │   └── BillStatus.java
    ├── dto/                                     # Data Transfer Object (Request/Response)
    │   ├── BillCreateRequest.java
    │   ├── BillUpdateRequest.java
    │   └── MessageResponse.java
    ├── repository/                              # Tầng Repository (Thao tác CSDL)
    │   └── BillRepository.java
    ├── service/                                 # Tầng Service (Xử lý nghiệp vụ logic)
    │   └── BillService.java
    ├── controller/                              # Tầng Controller (REST API Endpoints)
    │   └── BillController.java
    ├── security/                                # Tầng Bảo mật & Xác thực JWT
    │   ├── AuthUser.java
    │   └── JwtService.java
    └── exception/                               # Tầng Xử lý lỗi toàn cục
        ├── ApiException.java
        └── GlobalExceptionHandler.java
```

---

### 1. `pom.xml` (Cấu hình Maven Dependencies)
* **Ý nghĩa:** Khai báo thông tin dự án `com.pawpay:billing-service`, phiên bản Java 25 và các thư viện cần thiết.
* **Các thư viện chính:**
  * `spring-boot-starter-webmvc`: Xây dựng RESTful API.
  * `spring-boot-starter-validation`: Validate dữ liệu đầu vào (`@NotNull`, `@Positive`, `@NotBlank`).
  * `spring-boot-starter-data-jpa`: Tương tác với CSDL qua ORM Hibernate.
  * `h2`: CSDL in-memory chạy trực tiếp khi dev/test không cần cài đặt.
  * `mysql-connector-j`: Driver kết nối MySQL khi chạy Production/Docker.
  * `lombok`: Tự động sinh Getter, Setter, Builder, Constructor giúp code ngắn gọn, sạch sẽ.

---

### 2. `application.properties` (Cấu hình môi trường)
* **Ý nghĩa:** Chứa toàn bộ cấu hình chạy của service.
* **Chi tiết cấu hình:**
  * `server.port=3003`: Cổng chạy của Billing Service theo thỏa thuận nhóm.
  * `spring.jpa.open-in-view=false`: Tối ưu hiệu năng kết nối DB, tránh rò rỉ session.
  * `spring.datasource.url=jdbc:h2:mem:billingdb`: CSDL H2 in-memory chạy dev, tự tạo bảng tự động.
  * `jwt.secret=${JWT_SECRET:dev-secret-doi-khi-ghep-nhom}`: Khóa bí mật dùng chung giữa các microservices để giải mã và xác thực token JWT. Có thể ghi đè bằng biến môi trường khi deploy Docker.

---

### 3. `BillingServiceApplication.java` (Khởi chạy ứng dụng)
* **Ý nghĩa:** Điểm bắt đầu (Entry Point) của toàn bộ Microservice.
* **Chi tiết code:**
  * `@SpringBootApplication`: Kích hoạt tính năng Auto-Configuration, Component Scanning (`com.pawpay.billing.*`).
  * Hàm `main()` gọi `SpringApplication.run()` để start Web Server nhúng (Tomcat) trên port 3003.

---

### 4. Gói `entity` (Thực thể CSDL)

#### a. `Bill.java`
* **Ý nghĩa:** Đại diện cho bảng `bills` trong CSDL, chứa thông tin hóa đơn dịch vụ.
* **Các trường dữ liệu:**
  * `id`: Khóa chính tự tăng (`@Id @GeneratedValue(strategy = GenerationType.IDENTITY)`).
  * `petProfileId` (`pet_profile_id`): Mã hồ sơ thú cưng, liên kết logic sang Pet Service.
  * `serviceType` (`service_type`): Loại dịch vụ thú cưng (ví dụ: `spa`, `khám bệnh`, `tiêm phòng`).
  * `amount`: Số tiền thanh toán (kiểu `BigDecimal` với `precision = 15, scale = 2` chuẩn tài chính).
  * `status`: Trạng thái hóa đơn, kiểu Enum `BillStatus` (mặc định là `UNPAID`).
* **Clean Code:** Dùng `@JsonProperty` để tự động map giữa camelCase trong Java và snake_case (`pet_profile_id`, `service_type`) theo chuẩn API Contract.

#### b. `BillStatus.java`
* **Ý nghĩa:** Định nghĩa 2 trạng thái hợp lệ của hóa đơn: `UNPAID` ("unpaid") và `PAID` ("paid").
* **Điểm nổi bật:**
  * `@JsonValue`: Khi trả JSON ra client luôn ở dạng chữ thường (`"unpaid"`, `"paid"`).
  * `@JsonCreator` + `from(String text)`: Cho phép nhận cả chữ hoa lẫn chữ thường từ request (`unpaid`, `UNPAID`, `Paid`), nếu sai sẽ báo lỗi rõ ràng.

---

### 5. Gói `dto` (Data Transfer Object)

#### a. `BillCreateRequest.java`
* **Ý nghĩa:** DTO nhận dữ liệu khi Admin tạo mới một hóa đơn (`POST /bills`).
* **Validation:**
  * `petProfileId`: `@NotNull`, `@Positive(message = "pet_profile_id phải lớn hơn 0")`
  * `serviceType`: `@NotBlank(message = "Thiếu loại dịch vụ (service_type)")`
  * `amount`: `@NotNull`, `@Positive(message = "Số tiền phải lớn hơn 0")`

#### b. `BillUpdateRequest.java`
* **Ý nghĩa:** DTO nhận dữ liệu khi Admin cập nhật hóa đơn (`PUT /bills/{id}`).
* **Đặc điểm:** Toàn bộ các trường đều là tùy chọn (Optional). Client truyền trường nào thì cập nhật trường đó, không truyền thì giữ nguyên giá trị cũ.

#### c. `MessageResponse.java`
* **Ý nghĩa:** Record chứa phản hồi thông báo chuẩn `{ "message": "..." }`, đảm bảo Type-Safe thay vì dùng `Map<String, String>` rời rạc.

---

### 6. Gói `repository`

#### `BillRepository.java`
* **Ý nghĩa:** Giao tiếp với Database kế thừa từ `JpaRepository<Bill, Long>`.
* **Các hàm truy vấn:**
  * `findByPetProfileId(Long id)`: Tìm danh sách hóa đơn theo thú cưng.
  * `findByStatus(BillStatus status)`: Tìm danh sách hóa đơn theo trạng thái.
  * `findByPetProfileIdAndStatus(...)`: Kết hợp lọc cả 2 điều kiện.
  * **Hàm cốt lõi `markPaid`:**
    ```java
    @Modifying(clearAutomatically = true)
    @Query("update Bill b set b.status = :paid where b.id = :id and b.status = :unpaid")
    int markPaid(@Param("id") Long id, @Param("paid") BillStatus paid, @Param("unpaid") BillStatus unpaid);
    ```
    👉 **Ý nghĩa kỹ thuật cực kỳ quan trọng:** Đây là câu lệnh SQL Atomic có điều kiện. Nếu 2 request thanh toán gửi tới cùng 1 mili-giây, chỉ có đúng 1 request cập nhật được (trả về 1 dòng bị ảnh hưởng), request thứ hai trả về 0 dòng → Chống thanh toán 2 lần (Idempotency & Race Condition Prevention).

---

### 7. Gói `service`

#### `BillService.java`
* **Ý nghĩa:** Xử lý toàn bộ logic nghiệp vụ của Billing Service.
* **Các phương thức:**
  * `@Transactional(readOnly = true)` ở class level: Tối ưu hiệu năng truy vấn đọc.
  * `find(petProfileId, status)`: Xử lý linh hoạt 4 trường hợp tìm kiếm (theo pet, theo status, theo cả 2, hoặc xem tất cả).
  * `get(id)`: Lấy chi tiết hóa đơn, nếu không thấy tự động ném `ApiException(404, "Không tìm thấy hóa đơn")`.
  * `create(request)`: `@Transactional` tạo mới hóa đơn với trạng thái mặc định `UNPAID`.
  * `update(id, request)`: `@Transactional` cập nhật linh hoạt từng trường và kiểm tra rỗng.
  * `delete(id)`: `@Transactional` kiểm tra tồn tại và xóa.
  * `pay(id)`: Gọi `repo.markPaid()`. Nếu kết quả bằng 0 (hóa đơn đã thanh toán từ trước), ném ngay `ApiException(409, "Hóa đơn đã được thanh toán")`.

---

### 8. Gói `security`

#### a. `AuthUser.java`
* **Ý nghĩa:** Record lưu thông tin người dùng được giải mã từ JWT Token (`userId`, `role`). Có hàm tiện ích `isAdmin()` kiểm tra role không phân biệt hoa thường.

#### b. `JwtService.java`
* **Ý nghĩa:** Tự xác thực và giải mã JWT token chuẩn **HMAC-SHA256** bằng secret key chung của nhóm mà không cần phụ thuộc thư viện cồng kềnh.
* **Quy trình hoạt động:**
  1. Kiểm tra header `Authorization: Bearer <token>`.
  2. Tách Token thành 3 phần: `Header.Payload.Signature`.
  3. Dùng thuật toán `HmacSHA256` tính chữ ký kỳ vọng trên `Header.Payload` với `secret` key.
  4. So sánh chữ ký an toàn (`MessageDigest.isEqual`) để chống tấn công Timing Attack.
  5. Giải mã Base64 phần Payload, kiểm tra thời gian hết hạn (`exp`), trích xuất `user_id` và `role`.
  6. Phương thức `requireAdmin()` / `authenticateAdmin()` kiểm tra xem người dùng có quyền Admin hay không (ném lỗi `403` nếu không đủ quyền).

---

### 9. Gói `controller`

#### `BillController.java`
* **Ý nghĩa:** Tiếp nhận các HTTP Request từ client / Payment Service / Gateway và trả về JSON tương ứng.
* **Quy tắc phân quyền và danh sách 7 API Endpoints:**

| STT | HTTP Method | Đường dẫn | Phân quyền | Ý nghĩa nghiệp vụ | Mã HTTP trả về |
|:---:|:---:|:---|:---|:---|:---:|
| 1 | `GET` | `/bills` (không filter) | **Admin** | Xem toàn bộ hóa đơn trong hệ thống | 200 OK |
| 2 | `GET` | `/bills?pet_profile_id=1&status=unpaid` | User / Admin | Xem & lọc hóa đơn của thú cưng | 200 OK |
| 3 | `GET` | `/bills/{id}` | User / Admin | Xem chi tiết 1 hóa đơn (Payment gọi để check) | 200 OK |
| 4 | `POST` | `/bills` | **Admin** | Tạo mới hóa đơn dịch vụ (spa, khám bệnh) | 201 Created |
| 5 | `PUT` | `/bills/{id}` | **Admin** | Cập nhật thông tin hóa đơn | 200 OK |
| 6 | `DELETE` | `/bills/{id}` | **Admin** | Xóa hóa đơn | 200 OK |
| 7 | `PUT` | `/bills/{id}/pay` | User / Payment | Đánh dấu đã thanh toán (chỉ thành công 1 lần) | 200 OK / 409 Conflict |

---

### 10. Gói `exception`

#### a. `ApiException.java`
* Custom Runtime Exception mang theo mã HTTP Status (400, 401, 403, 404, 409, 500) và câu thông báo tiếng Việt thân thiện.

#### b. `GlobalExceptionHandler.java`
* Đánh dấu `@RestControllerAdvice` để bắt và chuyển đổi toàn bộ Exception xảy ra trong ứng dụng thành định dạng chuẩn `{ "message": "..." }` theo đúng API Contract nhóm:
  * `ApiException`: Trả về mã status và message tương ứng.
  * `MethodArgumentNotValidException`: Bắt lỗi validate DTO (ví dụ thiếu amount, pet_profile_id <= 0).
  * `HttpMessageNotReadableException`: Bắt lỗi JSON gửi sai định dạng hoặc enum status sai.
  * `MethodArgumentTypeMismatchException`: Bắt lỗi truyền param sai kiểu (ví dụ truyền chữ vào id số).
  * `HttpRequestMethodNotSupportedException`: Trả 405 khi gọi sai method (POST thay vì GET).
  * `Exception` (fallback): Ghi log lỗi bằng `@Slf4j` và trả mã 500 "Lỗi hệ thống".

---

## PHẦN 2: KỊCH BẢN THUYẾT TRÌNH TỪNG BƯỚC (CHO THẦY/CÔ)

Bạn có thể tự tin thuyết trình theo 5 bước mạch lạc sau:

### Bước 1: Mở đầu & Giới thiệu vai trò (30 giây)
> *"Kính thưa Thầy/Cô, trong đồ án PawPay Microservices, em là **Minh Sang**, đảm nhiệm xây dựng **Billing Service** (chạy tại port **3003** trên nền tảng **Java 25 và Spring Boot 4**).  
> Nhiệm vụ chính của dịch vụ này là quản lý toàn bộ danh mục dịch vụ và hóa đơn cần thanh toán (như dịch vụ spa, khám bệnh, tiêm chủng cho thú cưng), đồng thời cung cấp API nội bộ để dịch vụ Payment Service thực hiện gạch nợ hóa đơn khi người dùng thanh toán."*

### Bước 2: Trình bày sơ đồ Use Case (1 phút)
*(Mở file `docs/use-case-diagram.png` lên)*
> *"Trên sơ đồ Use Case, dịch vụ Billing Service có 3 tác nhân tương tác chính:
> 1. **Admin:** Có toàn quyền CRUD dịch vụ và hóa đơn (Tạo mới, Cập nhật, Xóa và Xem toàn bộ hóa đơn hệ thống).
> 2. **User (Khách hàng):** Tra cứu danh sách hóa đơn theo mã thú cưng (`pet_profile_id`) và lọc theo trạng thái `unpaid` hoặc `paid`.
> 3. **Payment Service (Hệ thống nội bộ):** Gọi API xem chi tiết hóa đơn để kiểm tra số tiền trước khi thanh toán và gọi API `PUT /bills/{id}/pay` để cập nhật trạng thái hóa đơn sau khi trừ tiền thành công.  
> Tất cả các use case của Admin và User đều có quan hệ `<<include>>` với module **Xác thực JWT**."*

### Bước 3: Trình bày cấu trúc Code & Điểm kỹ thuật nổi bật (2 phút)
> *"Về kiến trúc mã nguồn, em tổ chức dự án theo mô hình phân lớp rõ ràng:
> - **Entity & DTO:** Tách bạch giữa thực thể CSDL `Bill` và các DTO `BillCreateRequest`, `BillUpdateRequest` có gắn Bean Validation chặt chẽ.
> - **Xử lý bất đồng bộ & chống Race Condition:** Ở API thanh toán `PUT /bills/{id}/pay`, em sử dụng câu lệnh Atomic Update trực tiếp dưới database `WHERE id = :id AND status = 'unpaid'`. Điều này đảm bảo tính Idempotency: hóa đơn chỉ được thanh toán thành công duy nhất 1 lần, nếu có 2 giao dịch gọi đồng thời hoặc gọi lại lần hai sẽ lập tức trả về mã **409 Conflict**, ngăn chặn hoàn toàn rủi ro thanh toán trùng.
> - **Bảo mật JWT:** Em xây dựng `JwtService` tự verify chữ ký HMAC-SHA256 bằng secret key dùng chung của nhóm, kiểm tra thời hạn token `exp` và phân quyền linh hoạt (Admin được CRUD, User chỉ được xem hóa đơn của thú cưng).
> - **Global Exception Handler:** Bắt toàn bộ lỗi tập trung và trả về đúng format `{ "message": "..." }` theo quy ước chung của nhóm."*

### Bước 4: Demo nhanh các trường hợp (Nếu thầy yêu cầu test)
1. **Test Tạo hóa đơn (Admin):** `POST http://localhost:3003/bills` kèm Token Admin → Trả về **201 Created**.
2. **Test Xem hóa đơn theo thú cưng:** `GET http://localhost:3003/bills?pet_profile_id=1&status=unpaid` → Trả về mảng danh sách hóa đơn chưa thanh toán.
3. **Test Thanh toán lần 1:** `PUT http://localhost:3003/bills/1/pay` → Trả về **200 OK** (status chuyển thành `paid`).
4. **Test Thanh toán lần 2 (Test tính năng chống trả trùng):** `PUT http://localhost:3003/bills/1/pay` lần nữa → Trả về **409 Conflict** `{ "message": "Hóa đơn đã được thanh toán" }`.

---

## PHẦN 3: BỘ CÂU HỎI & CÂU TRẢ LỜI PHÒNG THỦ KHI THẦY HỎI (Q&A)

**❓ Câu 1: Tại sao em dùng `BigDecimal` cho trường `amount` mà không dùng `Double` hay `Float`?**
* **Trả lời:** *"Dạ thưa Thầy, trong các bài toán tài chính và hóa đơn, kiểu `Double` và `Float` bị lỗi làm tròn số thực dấu phẩy động (floating-point precision issue). Em sử dụng `BigDecimal` với định dạng `precision = 15, scale = 2` để đảm bảo độ chính xác tuyệt đối đến từng đồng tiền."*

**❓ Câu 2: Làm sao hệ thống của em ngăn được việc 2 người cùng bấm thanh toán 1 hóa đơn cùng lúc?**
* **Trả lời:** *"Dạ thưa Thầy, trong `BillRepository`, em không dùng cách đọc lên rồi mới kiểm tra bằng code Java (dễ bị race condition), mà em thực hiện câu lệnh Update nguyên tử có điều kiện: `UPDATE bills SET status = 'paid' WHERE id = :id AND status = 'unpaid'`. Cơ chế khóa mức dòng (Row-level Locking) của CSDL sẽ đảm bảo chỉ có đúng 1 request cập nhật được trạng thái, request còn lại thấy 0 dòng bị ảnh hưởng và Service sẽ ném ngay lỗi `409 Conflict`."*

**❓ Câu 3: JWT Token trong hệ thống được xác thực như thế nào giữa các service?**
* **Trả lời:** *"Dạ, các microservices trong hệ thống cùng chia sẻ một biến môi trường `JWT_SECRET`. Khi User đăng nhập ở User Service, token sẽ được ký bằng secret này. Khi client gửi request sang Billing Service kèm header `Authorization: Bearer <token>`, `JwtService` của em sẽ dùng thuật toán HMAC-SHA256 băm lại `Header.Payload` với `secret` rồi so khớp với phần Signature của token để xác thực tính toàn vẹn và trích xuất `role`, `user_id`."*

**❓ Câu 4: Nếu Database chính (MySQL) chưa khởi động thì dịch vụ có chạy test được không?**
* **Trả lời:** *"Dạ được, trong file `application.properties`, em đã cấu hình mặc định CSDL H2 in-memory (`jdbc:h2:mem:billingdb`). Khi chạy local để dev và test, hệ thống tự động khởi tạo CSDL trong RAM mà không cần cài MySQL. Khi triển khai Production, chỉ cần mở cấu hình MySQL qua các biến môi trường `DB_HOST`, `DB_NAME`, `DB_USER`."*
