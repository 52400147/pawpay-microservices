package com.pawpay.billing.exception;

/**
 * Custom runtime exception carrying an HTTP status code and user-friendly error message.
 */
public class ApiException extends RuntimeException {

    private final int status;

    public ApiException(int status, String message) {
        super(message);
        this.status = status;
    }

    public int getStatus() {
        return status;
    }
}
