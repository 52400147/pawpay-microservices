package com.pawpay.billing.security;

/**
 * Represents an authenticated user extracted from the JWT token.
 */
public record AuthUser(String userId, String role) {

    public boolean isAdmin() {
        return "admin".equalsIgnoreCase(role);
    }
}
