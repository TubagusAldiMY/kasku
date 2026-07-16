package tech.tubsamy.kasku.ui.debt

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.DebtItem
import tech.tubsamy.kasku.data.DebtPaymentItem
import tech.tubsamy.kasku.data.DebtsRepository

/**
 * Rincian satu debt: sisa/status + catat pembayaran + riwayat pembayaran.
 * debt dibaca dari cache repo (navigasi selalu dari daftar yang baru fetch); setelah
 * recordPayment repo.refresh() dipanggil → debt dibaca ulang untuk sisa/status terbaru.
 */
class DebtPaymentsViewModel(
    private val repo: DebtsRepository,
    private val debtId: String,
) : ViewModel() {

    var debt by mutableStateOf<DebtItem?>(null)
        private set
    var payments by mutableStateOf<List<DebtPaymentItem>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set

    var amount by mutableStateOf("") // digit string, rupiah
    var notes by mutableStateOf("")
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    val canRecord: Boolean
        get() = !saving && (amount.toLongOrNull() ?: 0L) > 0L && debt?.isSettled == false

    init {
        loadPayments()
    }

    private fun loadPayments() {
        viewModelScope.launch {
            loading = true
            debt = repo.debts.value.firstOrNull { it.id == debtId }
            payments = runCatching { repo.listPayments(debtId) }.getOrDefault(payments)
            loading = false
        }
    }

    fun recordPayment() {
        val amt = amount.toLongOrNull() ?: 0L
        if (!canRecord) return
        saving = true
        error = null
        viewModelScope.launch {
            try {
                repo.recordPayment(debtId, amt, notes)
                amount = ""
                notes = ""
                debt = repo.debts.value.firstOrNull { it.id == debtId }
                payments = repo.listPayments(debtId)
                saving = false
            } catch (e: Exception) {
                saving = false
                error = "Gagal mencatat pembayaran. Periksa koneksi atau jumlah melebihi sisa."
            }
        }
    }

    companion object {
        fun factory(
            repo: DebtsRepository,
            debtId: String,
        ) = viewModelFactory {
            initializer { DebtPaymentsViewModel(repo, debtId) }
        }
    }
}
