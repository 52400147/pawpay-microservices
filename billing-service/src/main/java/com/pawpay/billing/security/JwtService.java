package com.pawpay.billing.security;

import com.pawpay.billing.exception.ApiException;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import tools.jackson.databind.json.JsonMapper;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.Base64;
import java.util.Map;

/**
 * Service for verifying and decoding JWT tokens using shared HMAC-SHA256 secret.
 */
@Component
public class JwtService {

    private static final String ALGORITHM = "HmacSHA256";
    private static final String BEARER_PREFIX = "Bearer ";

    private final byte[] secret;
    private final JsonMapper mapper = JsonMapper.builder().build();

    public JwtService(@Value("${jwt.secret}") String secret) {
        this.secret = secret.getBytes(StandardCharsets.UTF_8);
    }

    /**
     * Authenticates the user from the Authorization header.
     *
     * @param authorizationHeader The Authorization header containing the Bearer token.
     * @return The authenticated user identity.
     * @throws ApiException if missing, invalid, or expired.
     */
    public AuthUser authenticate(String authorizationHeader) {
        if (authorizationHeader == null || !authorizationHeader.regionMatches(true, 0, BEARER_PREFIX, 0, BEARER_PREFIX.length())) {
            throw new ApiException(401, "Bạn chưa đăng nhập");
        }

        String token = authorizationHeader.substring(BEARER_PREFIX.length()).trim();
        String[] parts = token.split("\\.");
        if (parts.length != 3) {
            throw new ApiException(401, "Token không hợp lệ");
        }

        try {
            Mac mac = Mac.getInstance(ALGORITHM);
            mac.init(new SecretKeySpec(secret, ALGORITHM));
            byte[] expected = mac.doFinal((parts[0] + "." + parts[1]).getBytes(StandardCharsets.UTF_8));
            byte[] actual = Base64.getUrlDecoder().decode(parts[2]);

            if (!MessageDigest.isEqual(expected, actual)) {
                throw new ApiException(401, "Token không hợp lệ");
            }

            byte[] payloadJson = Base64.getUrlDecoder().decode(parts[1]);
            @SuppressWarnings("unchecked")
            Map<String, Object> payload = mapper.readValue(payloadJson, Map.class);

            Object exp = payload.get("exp");
            if (exp instanceof Number n && n.longValue() < System.currentTimeMillis() / 1000) {
                throw new ApiException(401, "Token đã hết hạn");
            }

            Object userId = payload.get("user_id");
            Object role = payload.get("role");
            if (userId == null || role == null) {
                throw new ApiException(401, "Token không hợp lệ");
            }

            return new AuthUser(String.valueOf(userId), String.valueOf(role));
        } catch (ApiException e) {
            throw e;
        } catch (Exception e) {
            throw new ApiException(401, "Token không hợp lệ");
        }
    }

    /**
     * Ensures that the authenticated user possesses the admin role.
     *
     * @param user Authenticated user.
     * @return The same user if admin.
     * @throws ApiException if user is not an admin.
     */
    public AuthUser requireAdmin(AuthUser user) {
        if (!user.isAdmin()) {
            throw new ApiException(403, "Bạn không có quyền thực hiện thao tác này");
        }
        return user;
    }

    /**
     * Convenience method to authenticate and verify admin role in one call.
     *
     * @param authorizationHeader The Authorization header.
     * @return The authenticated admin user.
     */
    public AuthUser authenticateAdmin(String authorizationHeader) {
        return requireAdmin(authenticate(authorizationHeader));
    }
}
