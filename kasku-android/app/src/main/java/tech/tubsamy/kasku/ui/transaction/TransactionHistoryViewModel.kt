package tech.tubsamy.kasku.ui.transaction

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.TransactionItem
import tech.tubsamy.kasku.data.TransactionsRepository

/**
 * F2 Riwayat — list transaksi dari cache TransactionsRepository (ONLINE, urut tanggal desc).
 * refresh() memuat kategori dulu supaya categoryName benar sejak emit pertama.
 */
class TransactionHistoryViewModel(
    private val transactions: TransactionsRepository,
) : ViewModel() {

    /** Hapus via REST; repo menyegarkan cache → list re-emit. */
    fun delete(id: String) {
        viewModelScope.launch { transactions.delete(id) }
    }

    /** Filter tipe (pills): null = Semua, "INCOME" = Masuk, "EXPENSE" = Keluar. */
    val typeFilter = MutableStateFlow<String?>(null)

    val items: StateFlow<List<TransactionItem>> =
        combine(transactions.transactions, typeFilter) { txs, type ->
            if (type == null) txs else txs.filter { it.type == type }
        }.stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = emptyList(),
        )

    init {
        viewModelScope.launch { transactions.refresh() }
    }

    companion object {
        fun factory(transactions: TransactionsRepository) = viewModelFactory {
            initializer { TransactionHistoryViewModel(transactions) }
        }
    }
}
