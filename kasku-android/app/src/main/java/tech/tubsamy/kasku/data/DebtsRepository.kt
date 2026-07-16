package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import tech.tubsamy.kasku.data.remote.DebtApi
import tech.tubsamy.kasku.data.remote.dto.CreateDebtRequest
import tech.tubsamy.kasku.data.remote.dto.DebtDto
import tech.tubsamy.kasku.data.remote.dto.RecordPaymentRequest
import tech.tubsamy.kasku.data.remote.dto.UpdateDebtRequest

/** Catatan hutang/piutang untuk UI (backend hitung sisa). */
data class DebtItem(
    val id: String,
    val direction: String, // RECEIVABLE (piutang) | PAYABLE (hutang)
    val personName: String,
    val totalAmount: Long,
    val remainingAmount: Long,
    val dueDate: String?, // "YYYY-MM-DD" atau null
    val notes: String,
    val status: String, // ACTIVE | SETTLED
) {
    val isReceivable: Boolean get() = direction == "RECEIVABLE"
    val isSettled: Boolean get() = status == "SETTLED"
    val paidAmount: Long get() = totalAmount - remainingAmount
}

data class DebtPaymentItem(
    val id: String,
    val amount: Long,
    val paymentDate: String, // "YYYY-MM-DD"
    val notes: String,
)

/**
 * Pola sama dengan BudgetsRepository: fetch online (GET /v1/debts) → cache in-memory
 * sebagai StateFlow. Gagal fetch → cache lama dipertahankan (tak throw). Mutasi CRUD +
 * pencatatan pembayaran → throw bila gagal (VM tampilkan error), sukses → refresh cache.
 */
class DebtsRepository(
    private val debtApi: DebtApi,
) {
    private val _debts = MutableStateFlow<List<DebtItem>>(emptyList())
    val debts: StateFlow<List<DebtItem>> = _debts

    /** Fetch ulang; offline/5xx → diam, nilai lama tetap tampil. */
    suspend fun refresh() {
        val loaded = runCatching {
            (debtApi.listDebts().data ?: emptyList()).map { it.toItem() }
        }.getOrNull() ?: return
        _debts.value = loaded
    }

    suspend fun create(direction: String, personName: String, amount: Long, dueDate: String?, notes: String) {
        debtApi.createDebt(
            CreateDebtRequest(
                direction = direction,
                personName = personName.trim(),
                amount = amount,
                dueDate = dueDate,
                notes = notes.trim(),
            ),
        )
        refresh()
    }

    suspend fun update(id: String, personName: String, dueDate: String?, notes: String) {
        debtApi.updateDebt(
            id,
            UpdateDebtRequest(
                personName = personName.trim(),
                dueDate = dueDate,
                notes = notes.trim(),
            ),
        )
        refresh()
    }

    suspend fun delete(id: String) {
        debtApi.deleteDebt(id)
        refresh()
    }

    /** Riwayat pembayaran satu debt — di-fetch on demand (tidak di-cache). */
    suspend fun listPayments(debtId: String): List<DebtPaymentItem> =
        (debtApi.listPayments(debtId).data ?: emptyList()).map {
            DebtPaymentItem(
                id = it.id,
                amount = it.amount,
                paymentDate = it.paymentDate.take(10),
                notes = it.notes,
            )
        }

    suspend fun recordPayment(debtId: String, amount: Long, notes: String) {
        debtApi.recordPayment(debtId, RecordPaymentRequest(amount = amount, notes = notes.trim()))
        refresh() // sisa/status debt berubah
    }
}

private fun DebtDto.toItem() = DebtItem(
    id = id,
    direction = direction,
    personName = personName,
    totalAmount = totalAmount,
    remainingAmount = remainingAmount,
    dueDate = dueDate?.takeIf { it.isNotBlank() }?.take(10), // RFC3339 → tanggal saja
    notes = notes,
    status = status,
)
