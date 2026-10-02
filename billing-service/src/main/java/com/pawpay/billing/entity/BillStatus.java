package com.pawpay.billing.entity;

import com.fasterxml.jackson.annotation.JsonCreator;
import com.fasterxml.jackson.annotation.JsonValue;

/**
 * Status of a bill (unpaid or paid).
 */
public enum BillStatus {
    UNPAID("unpaid"),
    PAID("paid");

    private final String value;

    BillStatus(String value) {
        this.value = value;
    }

    @JsonValue
    public String getValue() {
        return value;
    }

    @JsonCreator
    public static BillStatus from(String text) {
        if (text != null) {
            for (BillStatus s : values()) {
                if (s.value.equalsIgnoreCase(text.trim())) {
                    return s;
                }
            }
        }
        throw new IllegalArgumentException("status phải là unpaid hoặc paid");
    }
}
