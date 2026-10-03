package com.pawpay.billing.service;

import com.pawpay.billing.dto.BillCreateRequest;
import com.pawpay.billing.dto.BillUpdateRequest;
import com.pawpay.billing.entity.Bill;
import com.pawpay.billing.entity.BillStatus;
import com.pawpay.billing.exception.ApiException;
import com.pawpay.billing.repository.BillRepository;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;

@Service
@Transactional(readOnly = true)
public class BillService {

    private final BillRepository billRepository;

    public BillService(BillRepository billRepository) {
        this.billRepository = billRepository;
    }

    /**
     * Finds bills optionally filtered by petProfileId and/or status.
     *
     * @param petProfileId optional Pet Profile ID filter.
     * @param status       optional status filter string ("unpaid" or "paid").
     * @return List of matching bills.
     */
    public List<Bill> find(Long petProfileId, String status) {
        BillStatus billStatus = null;
        if (status != null && !status.isBlank()) {
            try {
                billStatus = BillStatus.from(status);
            } catch (IllegalArgumentException e) {
                throw new ApiException(400, "status phải là unpaid hoặc paid");
            }
        }

        if (petProfileId != null && billStatus != null) {
            return billRepository.findByPetProfileIdAndStatus(petProfileId, billStatus);
        }
        if (petProfileId != null) {
            return billRepository.findByPetProfileId(petProfileId);
        }
        if (billStatus != null) {
            return billRepository.findByStatus(billStatus);
        }
        return billRepository.findAll();
    }

    /**
     * Gets a single bill by ID.
     *
     * @param id Bill ID.
     * @return Found bill entity.
     * @throws ApiException (404) if not found.
     */
    public Bill get(Long id) {
        return billRepository.findById(id)
                .orElseThrow(() -> new ApiException(404, "Không tìm thấy hóa đơn"));
    }

    /**
     * Creates a new bill with UNPAID status by default.
     *
     * @param request Creation request DTO.
     * @return Saved bill entity.
     */
    @Transactional
    public Bill create(BillCreateRequest request) {
        Bill bill = Bill.builder()
                .petProfileId(request.petProfileId())
                .serviceType(request.serviceType().trim())
                .amount(request.amount())
                .status(request.status() != null ? request.status() : BillStatus.UNPAID)
                .build();
        return billRepository.save(bill);
    }

    /**
     * Updates an existing bill. Only non-null fields from request are modified.
     *
     * @param id      Bill ID to update.
     * @param request Update request DTO.
     * @return Updated bill entity.
     */
    @Transactional
    public Bill update(Long id, BillUpdateRequest request) {
        Bill bill = get(id);

        if (request.petProfileId() != null) {
            bill.setPetProfileId(request.petProfileId());
        }
        if (request.serviceType() != null) {
            if (request.serviceType().isBlank()) {
                throw new ApiException(400, "Loại dịch vụ không được để trống");
            }
            bill.setServiceType(request.serviceType().trim());
        }
        if (request.amount() != null) {
            bill.setAmount(request.amount());
        }
        if (request.status() != null) {
            bill.setStatus(request.status());
        }

        return billRepository.save(bill);
    }

    /**
     * Deletes a bill by ID.
     *
     * @param id Bill ID to delete.
     * @throws ApiException (404) if bill does not exist.
     */
    @Transactional
    public void delete(Long id) {
        if (!billRepository.existsById(id)) {
            throw new ApiException(404, "Không tìm thấy hóa đơn");
        }
        billRepository.deleteById(id);
    }

    /**
     * Atomically marks a bill as PAID. Prevents double-payment race conditions.
     *
     * @param id Bill ID to pay.
     * @return Updated bill entity.
     * @throws ApiException (404) if bill doesn't exist, or (409) if already paid.
     */
    @Transactional
    public Bill pay(Long id) {
        int updated = billRepository.markPaid(id, BillStatus.PAID, BillStatus.UNPAID);
        if (updated == 0) {
            if (!billRepository.existsById(id)) {
                throw new ApiException(404, "Không tìm thấy hóa đơn");
            }
            throw new ApiException(409, "Hóa đơn đã được thanh toán");
        }
        return get(id);
    }
}
