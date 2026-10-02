// PawPay – Payment/Transaction Service (port 3004)
//
// API (theo Apicontract.docx):
//
//	POST /payments               { bill_id, pet_profile_id }  -> 201 { transaction_id }
//	POST /payments/{id}/confirm  { otp }                      -> 200
//	GET  /transactions           (token)                      -> giao dịch của người đăng nhập
//	GET  /admin/transactions     (token admin)                -> toàn bộ giao dịch
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

// ---------------------------------------------------------------- cấu hình

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   []byte
	UserURL     string
	BillingURL  string
	OTPURL      string
	MaxOTPTry   int
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func loadConfig() Config {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("thiếu biến môi trường JWT_SECRET (dùng chung cả nhóm)")
	}
	return Config{
		Port:        env("PORT", "3004"),
		DatabaseURL: env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pawpay_payment?sslmode=disable"),
		JWTSecret:   []byte(secret),
		UserURL:     strings.TrimRight(env("USER_SERVICE_URL", "http://localhost:3001"), "/"),
		BillingURL:  strings.TrimRight(env("BILLING_SERVICE_URL", "http://localhost:3003"), "/"),
		OTPURL:      strings.TrimRight(env("OTP_SERVICE_URL", "http://localhost:3005"), "/"),
		MaxOTPTry:   5,
	}
}

// ---------------------------------------------------------------- model

const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

// Transaction khớp bảng transactions trong hợp đồng API.
type Transaction struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	BillID    int64     `json:"bill_id"`
	Amount    int64     `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ---------------------------------------------------------------- lỗi

// APIError: lỗi nghiệp vụ, trả về đúng { "message": ... } + HTTP status.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return e.Message }

func apiErr(status int, msg string) *APIError { return &APIError{status, msg} }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	var ae *APIError
	if errors.As(err, &ae) {
		writeJSON(w, ae.Status, map[string]string{"message": ae.Message})
		return
	}
	log.Printf("lỗi nội bộ: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Lỗi hệ thống, vui lòng thử lại sau"})
}

// ---------------------------------------------------------------- xác thực JWT

type Claims struct {
	UserID int64
	Role   string
	Token  string // token gốc, để chuyển tiếp cho User Service (GET /users/me)
}

type ctxKey struct{}

func (s *Server) parseToken(r *http.Request) (*Claims, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, apiErr(401, "Bạn chưa đăng nhập")
	}
	raw := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("thuật toán không hợp lệ")
		}
		return s.cfg.JWTSecret, nil
	})
	if err != nil || !tok.Valid {
		return nil, apiErr(401, "Token không hợp lệ hoặc đã hết hạn")
	}
	m, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, apiErr(401, "Token không hợp lệ")
	}
	var uid int64
	switch v := m["user_id"].(type) {
	case float64:
		uid = int64(v)
	case string:
		uid, _ = strconv.ParseInt(v, 10, 64)
	}
	role, _ := m["role"].(string)
	if uid == 0 {
		return nil, apiErr(401, "Token thiếu user_id")
	}
	return &Claims{UserID: uid, Role: role, Token: raw}, nil
}

func (s *Server) auth(adminOnly bool, next func(http.ResponseWriter, *http.Request, *Claims)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := s.parseToken(r)
		if err != nil {
			writeErr(w, err)
			return
		}
		if adminOnly && c.Role != "admin" {
			writeErr(w, apiErr(403, "Bạn không có quyền thực hiện thao tác này"))
			return
		}
		next(w, r, c)
	}
}

// serviceToken: token nội bộ để Payment gọi các API nội bộ của service khác
// (cùng JWT_SECRET, role admin). Hết hạn sau 1 phút.
func (s *Server) serviceToken() string {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 1000000, // id giả cho service, không trùng user thật
		"role":    "admin",
		"service": "payment",
		"exp":     time.Now().Add(time.Minute).Unix(),
	})
	str, _ := t.SignedString(s.cfg.JWTSecret)
	return str
}

// ---------------------------------------------------------------- client gọi service khác

type Server struct {
	cfg  Config
	db   *sql.DB
	http *http.Client
}

// call gọi service khác; trả (status, body). Lỗi mạng -> 502.
func (s *Server) call(ctx context.Context, method, u, token string, body any) (int, []byte, error) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return 0, nil, apiErr(502, "Không kết nối được tới dịch vụ liên quan, vui lòng thử lại")
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, b, nil
}

// unwrap: service có thể trả trực tiếp hoặc bọc trong {"data": ...}
func unwrap(b []byte) json.RawMessage {
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(b, &env) == nil && len(env.Data) > 0 && string(env.Data) != "null" {
		return env.Data
	}
	return b
}

func remoteMsg(b []byte, def string) string {
	var m struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(b, &m) == nil && m.Message != "" {
		return m.Message
	}
	return def
}

// num đọc số nguyên từ JSON (chấp nhận 150000, 150000.0 hoặc "150000").
type num int64

func (n *num) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*n = num(math.Round(f))
	return nil
}

type Bill struct {
	ID           int64  `json:"id"`
	PetProfileID int64  `json:"pet_profile_id"`
	ServiceType  string `json:"service_type"`
	Amount       num    `json:"amount"`
	Status       string `json:"status"`
}

type UserInfo struct {
	ID      int64  `json:"id"`
	Email   string `json:"email"`
	Balance num    `json:"balance"`
}

// getBill lấy hóa đơn theo id thông qua GET /bills?pet_profile_id=..  (không lọc status
// để phân biệt 404 và 409).
func (s *Server) getBill(ctx context.Context, billID, petID int64) (*Bill, error) {
	q := url.Values{"pet_profile_id": {strconv.FormatInt(petID, 10)}}
	status, body, err := s.call(ctx, "GET", s.cfg.BillingURL+"/bills?"+q.Encode(), s.serviceToken(), nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, apiErr(502, remoteMsg(body, "Không lấy được thông tin hóa đơn"))
	}
	var bills []Bill
	if err := json.Unmarshal(unwrap(body), &bills); err != nil {
		return nil, apiErr(502, "Dữ liệu hóa đơn không hợp lệ")
	}
	for i := range bills {
		if bills[i].ID == billID {
			return &bills[i], nil
		}
	}
	return nil, apiErr(404, "Không tìm thấy hóa đơn")
}

func (s *Server) getUser(ctx context.Context, token string) (*UserInfo, error) {
	status, body, err := s.call(ctx, "GET", s.cfg.UserURL+"/users/me", token, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, apiErr(502, remoteMsg(body, "Không lấy được thông tin người dùng"))
	}
	var u UserInfo
	if err := json.Unmarshal(unwrap(body), &u); err != nil {
		return nil, apiErr(502, "Dữ liệu người dùng không hợp lệ")
	}
	return &u, nil
}

// ---------------------------------------------------------------- handlers

type createReq struct {
	BillID       int64 `json:"bill_id"`
	PetProfileID int64 `json:"pet_profile_id"`
}

// POST /payments
func (s *Server) createPayment(w http.ResponseWriter, r *http.Request, c *Claims) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BillID <= 0 || req.PetProfileID <= 0 {
		writeErr(w, apiErr(400, "Thiếu hoặc sai bill_id / pet_profile_id"))
		return
	}
	ctx := r.Context()

	bill, err := s.getBill(ctx, req.BillID, req.PetProfileID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if bill.Status != "unpaid" {
		writeErr(w, apiErr(409, "Hóa đơn đã được thanh toán"))
		return
	}
	user, err := s.getUser(ctx, c.Token)
	if err != nil {
		writeErr(w, err)
		return
	}
	if int64(user.Balance) < int64(bill.Amount) {
		writeErr(w, apiErr(422, "Số dư không đủ để thanh toán hóa đơn này"))
		return
	}

	var txID int64
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO transactions (user_id, bill_id, amount, status)
		 VALUES ($1,$2,$3,'pending') RETURNING id`,
		c.UserID, bill.ID, int64(bill.Amount)).Scan(&txID)
	if err != nil {
		writeErr(w, err)
		return
	}

	// Gọi OTP Service gửi mã. Lỗi -> đánh dấu giao dịch failed.
	status, body, err := s.call(ctx, "POST", s.cfg.OTPURL+"/otp/generate", s.serviceToken(),
		map[string]any{"transaction_id": txID, "email": user.Email})
	if err != nil || status/100 != 2 {
		s.setStatus(txID, StatusFailed)
		msg := "Không gửi được mã OTP, vui lòng thử lại"
		if err == nil {
			msg = remoteMsg(body, msg)
		}
		writeErr(w, apiErr(502, msg))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"transaction_id": txID})
}

func (s *Server) setStatus(id int64, status string) {
	if _, err := s.db.Exec(`UPDATE transactions SET status=$1 WHERE id=$2 AND status='pending'`, status, id); err != nil {
		log.Printf("không cập nhật được trạng thái giao dịch %d: %v", id, err)
	}
}

type confirmReq struct {
	OTP string `json:"otp"`
}

// POST /payments/{id}/confirm
func (s *Server) confirmPayment(w http.ResponseWriter, r *http.Request, c *Claims) {
	txID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || txID <= 0 {
		writeErr(w, apiErr(400, "Mã giao dịch không hợp lệ"))
		return
	}
	var req confirmReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.OTP) == "" {
		writeErr(w, apiErr(400, "Vui lòng nhập mã OTP"))
		return
	}
	if err := s.doConfirm(r.Context(), txID, strings.TrimSpace(req.OTP), c); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Thanh toán thành công", "transaction_id": txID})
}

// doConfirm: toàn bộ luồng xác nhận trong MỘT DB transaction với khóa:
//  1. SELECT ... FOR UPDATE trên dòng giao dịch  -> chặn confirm trùng cùng một giao dịch
//  2. pg_advisory_xact_lock(bill)                -> chặn 2 người trả cùng 1 hóa đơn
//  3. pg_advisory_xact_lock(user)                -> chặn trừ số dư song song của cùng 1 user
//
// Thứ tự khóa cố định (giao dịch -> hóa đơn -> user) để tránh deadlock.
// Số dư và hóa đơn nằm ở DB của service khác nên Payment không thể SELECT FOR UPDATE trực tiếp;
// advisory lock đóng vai trò khóa logic, còn deduct/pay phía service kia vẫn phải atomic.
func (s *Server) doConfirm(ctx context.Context, txID int64, otp string, c *Claims) (err error) {
	dbtx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = dbtx.Rollback()
		}
	}()

	// (1) khóa dòng giao dịch
	var t Transaction
	var attempts int
	err = dbtx.QueryRowContext(ctx,
		`SELECT id, user_id, bill_id, amount, status, created_at, otp_attempts
		   FROM transactions WHERE id=$1 AND user_id=$2 FOR UPDATE`, txID, c.UserID).
		Scan(&t.ID, &t.UserID, &t.BillID, &t.Amount, &t.Status, &t.CreatedAt, &attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return apiErr(404, "Không tìm thấy giao dịch")
	}
	if err != nil {
		return err
	}
	if t.Status != StatusPending {
		return apiErr(409, "Giao dịch này đã được xử lý")
	}

	// (2)(3) khóa logic hóa đơn và người dùng
	if _, err = dbtx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(1, $1)`, int32(t.BillID)); err != nil {
		return err
	}
	if _, err = dbtx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(2, $1)`, int32(t.UserID)); err != nil {
		return err
	}

	// fail đánh dấu giao dịch thất bại và COMMIT (giữ lịch sử), rồi trả lỗi cho client.
	fail := func(e *APIError) error {
		if _, err := dbtx.ExecContext(ctx, `UPDATE transactions SET status='failed' WHERE id=$1`, t.ID); err != nil {
			return err
		}
		if err := dbtx.Commit(); err != nil {
			return err
		}
		committed = true
		return e
	}

	// hóa đơn này đã có giao dịch thành công (do chính Payment ghi nhận) -> không trừ tiền nữa
	var paidBefore bool
	if err = dbtx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM transactions WHERE bill_id=$1 AND status='success')`, t.BillID).
		Scan(&paidBefore); err != nil {
		return err
	}
	if paidBefore {
		return fail(apiErr(409, "Hóa đơn đã được người khác thanh toán"))
	}

	// xác thực OTP (OTP Service đảm bảo dùng một lần, hết hạn 5 phút)
	st, body, cerr := s.call(ctx, "POST", s.cfg.OTPURL+"/otp/verify", s.serviceToken(),
		map[string]any{"transaction_id": t.ID, "code": otp})
	if cerr != nil {
		return cerr // lỗi mạng: giữ pending để thử lại
	}
	if st != 200 {
		attempts++
		if attempts >= s.cfg.MaxOTPTry {
			return fail(apiErr(400, "Nhập sai OTP quá nhiều lần, giao dịch đã bị hủy"))
		}
		if _, err = dbtx.ExecContext(ctx, `UPDATE transactions SET otp_attempts=$1 WHERE id=$2`, attempts, t.ID); err != nil {
			return err
		}
		if err = dbtx.Commit(); err != nil {
			return err
		}
		committed = true
		return apiErr(400, remoteMsg(body, "Mã OTP sai hoặc đã hết hạn"))
	}

	// kiểm tra lại hóa đơn và số dư
	// pet_profile_id không lưu trong bảng nên lấy lại từ Billing bằng bill_id
	bill, berr := s.getBillByID(ctx, t.BillID)
	if berr != nil {
		return berr
	}
	if bill.Status != "unpaid" {
		return fail(apiErr(409, "Hóa đơn đã được người khác thanh toán"))
	}
	user, uerr := s.getUser(ctx, c.Token)
	if uerr != nil {
		return uerr
	}
	if int64(user.Balance) < t.Amount {
		return fail(apiErr(422, "Số dư không đủ để thanh toán"))
	}

	// trừ số dư (User Service)
	st, body, cerr = s.call(ctx, "POST", fmt.Sprintf("%s/users/%d/deduct", s.cfg.UserURL, t.UserID),
		s.serviceToken(), map[string]any{"amount": t.Amount})
	if cerr != nil {
		return cerr
	}
	switch {
	case st == 422 || st == 400:
		return fail(apiErr(422, remoteMsg(body, "Số dư không đủ để thanh toán")))
	case st/100 != 2:
		return apiErr(502, remoteMsg(body, "Không trừ được số dư"))
	}

	// đánh dấu hóa đơn đã thanh toán (Billing Service, chỉ thành công 1 lần)
	st, body, cerr = s.call(ctx, "PUT", fmt.Sprintf("%s/bills/%d/pay", s.cfg.BillingURL, t.BillID),
		s.serviceToken(), nil)
	if cerr != nil || st/100 != 2 {
		// đã trừ tiền mà hóa đơn không được đánh dấu -> hoàn tiền (bù trừ)
		s.refund(ctx, t.UserID, t.Amount)
		if cerr != nil {
			return fail(apiErr(502, "Không cập nhật được hóa đơn, tiền đã được hoàn lại"))
		}
		if st == 409 {
			return fail(apiErr(409, "Hóa đơn đã được người khác thanh toán, tiền đã được hoàn lại"))
		}
		return fail(apiErr(502, remoteMsg(body, "Không cập nhật được hóa đơn, tiền đã được hoàn lại")))
	}

	// ghi nhận thành công
	if _, err = dbtx.ExecContext(ctx, `UPDATE transactions SET status='success' WHERE id=$1`, t.ID); err != nil {
		// DB lỗi sau khi đã trừ tiền + paid: hoàn tác phía service khác không khả thi, ghi log để xử lý tay
		log.Printf("CẢNH BÁO: giao dịch %d đã trừ tiền và paid nhưng không ghi được success: %v", t.ID, err)
		return err
	}
	if err = dbtx.Commit(); err != nil {
		log.Printf("CẢNH BÁO: giao dịch %d commit lỗi sau khi đã trừ tiền: %v", t.ID, err)
		return err
	}
	committed = true

	// gửi email xác nhận: lỗi ở đây không làm hỏng giao dịch đã thành công
	go func() {
		c2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _, _ = s.call(c2, "POST", s.cfg.OTPURL+"/notifications/success", s.serviceToken(),
			map[string]any{
				"email":          user.Email,
				"transaction_id": t.ID,
				"bill_id":        t.BillID,
				"amount":         t.Amount,
			})
	}()
	return nil
}

// getBillByID: lấy hóa đơn theo id. Dùng GET /bills (admin) rồi lọc; nếu Billing có
// GET /bills/{id} thì đổi sang gọi trực tiếp cho nhanh.
func (s *Server) getBillByID(ctx context.Context, billID int64) (*Bill, error) {
	status, body, err := s.call(ctx, "GET", s.cfg.BillingURL+"/bills", s.serviceToken(), nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, apiErr(502, remoteMsg(body, "Không lấy được thông tin hóa đơn"))
	}
	var bills []Bill
	if err := json.Unmarshal(unwrap(body), &bills); err != nil {
		return nil, apiErr(502, "Dữ liệu hóa đơn không hợp lệ")
	}
	for i := range bills {
		if bills[i].ID == billID {
			return &bills[i], nil
		}
	}
	return nil, apiErr(404, "Không tìm thấy hóa đơn")
}

// refund hoàn tiền khi bù trừ; thử tối đa 3 lần.
func (s *Server) refund(ctx context.Context, userID, amount int64) {
	for i := 0; i < 3; i++ {
		st, _, err := s.call(ctx, "POST", fmt.Sprintf("%s/users/%d/refund", s.cfg.UserURL, userID),
			s.serviceToken(), map[string]any{"amount": amount})
		if err == nil && st/100 == 2 {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	log.Printf("CẢNH BÁO: hoàn tiền thất bại user=%d amount=%d – cần xử lý thủ công", userID, amount)
}

// GET /transactions – của người đang đăng nhập
func (s *Server) listMine(w http.ResponseWriter, r *http.Request, c *Claims) {
	s.list(w, r, `WHERE user_id=$1`, c.UserID)
}

// GET /admin/transactions – toàn bộ (lọc tùy chọn ?status=&user_id=)
func (s *Server) listAll(w http.ResponseWriter, r *http.Request, c *Claims) {
	where := []string{}
	args := []any{}
	if v := r.URL.Query().Get("status"); v != "" {
		if v != StatusPending && v != StatusSuccess && v != StatusFailed {
			writeErr(w, apiErr(400, "status phải là pending, success hoặc failed"))
			return
		}
		args = append(args, v)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}
	if v := r.URL.Query().Get("user_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeErr(w, apiErr(400, "user_id không hợp lệ"))
			return
		}
		args = append(args, id)
		where = append(where, fmt.Sprintf("user_id=$%d", len(args)))
	}
	clause := ""
	if len(where) > 0 {
		clause = "WHERE " + strings.Join(where, " AND ")
	}
	s.list(w, r, clause, args...)
}

func (s *Server) list(w http.ResponseWriter, r *http.Request, where string, args ...any) {
	rows, err := s.db.QueryContext(r.Context(),
		`SELECT id, user_id, bill_id, amount, status, created_at FROM transactions `+where+` ORDER BY id DESC`, args...)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer rows.Close()
	out := []Transaction{} // mảng rỗng thay vì null
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.BillID, &t.Amount, &t.Status, &t.CreatedAt); err != nil {
			writeErr(w, err)
			return
		}
		out = append(out, t)
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------------------------------------------------------------- main

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /payments", s.auth(false, s.createPayment))
	mux.HandleFunc("POST /payments/{id}/confirm", s.auth(false, s.confirmPayment))
	mux.HandleFunc("GET /transactions", s.auth(false, s.listMine))
	mux.HandleFunc("GET /admin/transactions", s.auth(true, s.listAll))
	return mux
}

func main() {
	cfg := loadConfig()
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		log.Fatalf("không kết nối được PostgreSQL: %v", err)
	}
	if schema, err := os.ReadFile("schema.sql"); err == nil {
		if _, err := db.Exec(string(schema)); err != nil {
			log.Fatalf("lỗi tạo bảng: %v", err)
		}
	}
	srv := &Server{cfg: cfg, db: db, http: &http.Client{Timeout: 8 * time.Second}}
	log.Printf("Payment Service chạy ở cổng %s", cfg.Port)
	log.Fatal((&http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}).ListenAndServe())
}
