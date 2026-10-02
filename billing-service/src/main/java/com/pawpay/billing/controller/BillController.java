package com.pawpay.billing.controller;

import com.pawpay.billing.dto.BillCreateRequest;
import com.pawpay.billing.dto.BillUpdateRequest;
import com.pawpay.billing.dto.MessageResponse;
import com.pawpay.billing.entity.Bill;
import com.pawpay.billing.security.JwtService;
import com.pawpay.billing.service.BillService;
import jakarta.validation.Valid;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

/**
 * REST controller exposing billing API endpoints.
 * <p>
 * Access rules:
 * - GET /bills (no filter)    → Admin only
 * - GET /bills?...            → Any authenticated user
 * - POST, PUT, DELETE         → Admin only
 * - PUT /{id}/pay             → Any authenticated user (called internally by Payment Service)
 */
@RestController
@RequestMapping("/bills")
public class BillController {

    private final BillService billService;
    private final JwtService jwtService;

    public BillController(BillService billService, JwtService jwtService) {
        this.billService = billService;
        this.jwtService = jwtService;
    }

    /**
     * Lists bills, optionally filtered by pet_profile_id and/or status.
     * Viewing ALL bills requires Admin token.
     */
    @GetMapping
    public List<Bill> list(
            @RequestHeader(value = "Authorization", required = false) String auth,
            @RequestParam(name = "pet_profile_id", required = false) Long petProfileId,
            @RequestParam(name = "status", required = false) String status) {

        var user = jwtService.authenticate(auth);
        if (petProfileId == null && (status == null || status.isBlank())) {
            jwtService.requireAdmin(user);
        }
        return billService.find(petProfileId, status);
    }

    /**
     * Gets a single bill by ID.
     * Used internally by Payment Service to verify a bill before charging.
     */
    @GetMapping("/{id}")
    public Bill get(
            @RequestHeader(value = "Authorization", required = false) String auth,
            @PathVariable Long id) {

        jwtService.authenticate(auth);
        return billService.get(id);
    }

    /**
     * Creates a new bill. Admin only.
     */
    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public Bill create(
            @RequestHeader(value = "Authorization", required = false) String auth,
            @Valid @RequestBody BillCreateRequest request) {

        jwtService.authenticateAdmin(auth);
        return billService.create(request);
    }

    /**
     * Updates an existing bill. Admin only.
     */
    @PutMapping("/{id}")
    public Bill update(
            @RequestHeader(value = "Authorization", required = false) String auth,
            @PathVariable Long id,
            @Valid @RequestBody BillUpdateRequest request) {

        jwtService.authenticateAdmin(auth);
        return billService.update(id, request);
    }

    /**
     * Deletes a bill by ID. Admin only.
     */
    @DeleteMapping("/{id}")
    public MessageResponse delete(
            @RequestHeader(value = "Authorization", required = false) String auth,
            @PathVariable Long id) {

        jwtService.authenticateAdmin(auth);
        billService.delete(id);
        return new MessageResponse("Đã xóa hóa đơn");
    }

    /**
     * Marks a bill as PAID. Idempotent-safe: only succeeds once.
     * Called internally by Payment Service after a successful transaction.
     * Returns 409 if the bill was already paid.
     */
    @PutMapping("/{id}/pay")
    public Bill pay(
            @RequestHeader(value = "Authorization", required = false) String auth,
            @PathVariable Long id) {

        jwtService.authenticate(auth);
        return billService.pay(id);
    }
}
