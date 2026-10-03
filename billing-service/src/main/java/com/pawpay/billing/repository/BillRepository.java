package com.pawpay.billing.repository;

import com.pawpay.billing.entity.Bill;
import com.pawpay.billing.entity.BillStatus;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;

@Repository
public interface BillRepository extends JpaRepository<Bill, Long> {

    List<Bill> findByPetProfileId(Long petProfileId);

    List<Bill> findByStatus(BillStatus status);

    List<Bill> findByPetProfileIdAndStatus(Long petProfileId, BillStatus status);

    /**
     * Atomically transitions bill status from UNPAID to PAID.
     *
     * @return Number of rows affected (1 if successful, 0 if bill does not exist or was already paid).
     */
    @Modifying(clearAutomatically = true)
    @Query("update Bill b set b.status = :paid where b.id = :id and b.status = :unpaid")
    int markPaid(@Param("id") Long id,
                 @Param("paid") BillStatus paid,
                 @Param("unpaid") BillStatus unpaid);
}
