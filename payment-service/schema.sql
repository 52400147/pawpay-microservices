-- PawPay – Payment Service (PostgreSQL)
CREATE TABLE IF NOT EXISTS transactions (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT      NOT NULL,
    bill_id      BIGINT      NOT NULL,
    amount       BIGINT      NOT NULL CHECK (amount > 0),          -- VND
    status       VARCHAR(10) NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending','success','failed')),
    otp_attempts INT         NOT NULL DEFAULT 0,                   -- số lần nhập sai OTP
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user   ON transactions (user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_bill   ON transactions (bill_id);
-- Mỗi hóa đơn chỉ được có TỐI ĐA MỘT giao dịch thành công (chốt chặn cuối cùng ở tầng DB)
CREATE UNIQUE INDEX IF NOT EXISTS uq_transactions_bill_success
    ON transactions (bill_id) WHERE status = 'success';
