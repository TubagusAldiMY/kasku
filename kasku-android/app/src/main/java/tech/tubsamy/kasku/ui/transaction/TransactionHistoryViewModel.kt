package tech.tubsamy.kasku.ui.transaction

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.CategoriesRepository
import tech.tubsamy.kasku.data.TransactionItem
import tech.tubsamy.kasku.data.TransactionsRepository
import tech.tubsamy.kasku.data.sync.TransactionMutations

/**
 * F2 Riwayat — list transaksi dari Room (urut kronologis desc). Nama kategori via
 * CategoriesRepository (cache in-memory); ensureLoaded() di init memicu re-emit list
 * dengan nama benar begitu cache datang (via combine categories.version di repo).
 */
class TransactionHistoryViewModel(
    transactions: TransactionsRepository,
    private val categories: CategoriesRepository,
    private val mutations: TransactionMutations,
) : ViewModel() {

    /** Hapus optimistic: soft-delete Room + antre sync. List re-emit sendiri (deleted=0 difilter). */
    fun delete(id: String) {
        viewModelScope.launch { mutations.delete(id) }
    }

    val items: StateFlow<List<TransactionItem>> =
        transactions.observeAll().stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = emptyList(),
        )

    init {
        viewModelScope.launch { categories.ensureLoaded() }
    }

    companion object {
        fun factory(
            transactions: TransactionsRepository,
            categories: CategoriesRepository,
            mutations: TransactionMutations,
        ) = viewModelFactory {
            initializer { TransactionHistoryViewModel(transactions, categories, mutations) }
        }
    }
}
