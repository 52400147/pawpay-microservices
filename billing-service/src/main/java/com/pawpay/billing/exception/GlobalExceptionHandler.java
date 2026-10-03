package com.pawpay.billing.exception;

import com.pawpay.billing.dto.MessageResponse;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.support.DefaultMessageSourceResolvable;
import org.springframework.http.ResponseEntity;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.web.HttpRequestMethodNotSupportedException;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException;
import org.springframework.web.servlet.resource.NoResourceFoundException;

import java.util.stream.Collectors;

/**
 * Global exception handler converting exceptions into standard JSON error format:
 * { "message": "..." } as required by the API Contract.
 */
@Slf4j
@RestControllerAdvice
public class GlobalExceptionHandler {

    private ResponseEntity<MessageResponse> response(int status, String message) {
        return ResponseEntity.status(status).body(new MessageResponse(message));
    }

    @ExceptionHandler(ApiException.class)
    public ResponseEntity<MessageResponse> handleApiException(ApiException e) {
        return response(e.getStatus(), e.getMessage());
    }

    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ResponseEntity<MessageResponse> handleValidationException(MethodArgumentNotValidException e) {
        String msg = e.getBindingResult().getFieldErrors().stream()
                .map(DefaultMessageSourceResolvable::getDefaultMessage)
                .collect(Collectors.joining("; "));
        return response(400, (msg == null || msg.isBlank()) ? "Dữ liệu không hợp lệ" : msg);
    }

    @ExceptionHandler(HttpMessageNotReadableException.class)
    public ResponseEntity<MessageResponse> handleUnreadableException(HttpMessageNotReadableException e) {
        return response(400, "Dữ liệu gửi lên không hợp lệ (kiểm tra JSON và trường status: unpaid hoặc paid)");
    }

    @ExceptionHandler(MethodArgumentTypeMismatchException.class)
    public ResponseEntity<MessageResponse> handleTypeMismatchException(MethodArgumentTypeMismatchException e) {
        return response(400, "Tham số '" + e.getName() + "' không hợp lệ");
    }

    @ExceptionHandler(IllegalArgumentException.class)
    public ResponseEntity<MessageResponse> handleIllegalArgumentException(IllegalArgumentException e) {
        return response(400, e.getMessage());
    }

    @ExceptionHandler(NoResourceFoundException.class)
    public ResponseEntity<MessageResponse> handleNoResourceException(NoResourceFoundException e) {
        return response(404, "Không tìm thấy đường dẫn");
    }

    @ExceptionHandler(HttpRequestMethodNotSupportedException.class)
    public ResponseEntity<MessageResponse> handleMethodNotSupportedException(HttpRequestMethodNotSupportedException e) {
        return response(405, "Phương thức không được hỗ trợ");
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<MessageResponse> handleGeneralException(Exception e) {
        log.error("Unhandled server exception: ", e);
        return response(500, "Lỗi hệ thống");
    }
}
