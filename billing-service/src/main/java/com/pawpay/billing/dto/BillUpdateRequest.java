package com.pawpay.billing.dto;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.pawpay.billing.entity.BillStatus;
import jakarta.validation.constraints.Positive;

import java.math.BigDecimal;

/**
 * Request payload for updating an existing bill.
 * All fields are optional: omitted fields remain unchanged.
 */
public record BillUpdateRequest(
        @JsonProperty("pet_profile_id")
        @Positive(message = "pet_profile_id phải lớn hơn 0")
        Long petProfileId,

        @JsonProperty("service_type")
        String serviceType,

        @Positive(message = "Số tiền phải lớn hơn 0")
        BigDecimal amount,

        BillStatus status
) {
}
