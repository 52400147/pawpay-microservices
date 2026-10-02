package com.pawpay.billing.dto;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.pawpay.billing.entity.BillStatus;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;

import java.math.BigDecimal;

/**
 * Request payload for creating a new bill.
 */
public record BillCreateRequest(
        @JsonProperty("pet_profile_id")
        @NotNull(message = "Thiếu pet_profile_id")
        @Positive(message = "pet_profile_id phải lớn hơn 0")
        Long petProfileId,

        @JsonProperty("service_type")
        @NotBlank(message = "Thiếu loại dịch vụ (service_type)")
        String serviceType,

        @NotNull(message = "Thiếu số tiền (amount)")
        @Positive(message = "Số tiền phải lớn hơn 0")
        BigDecimal amount,

        BillStatus status
) {
}
