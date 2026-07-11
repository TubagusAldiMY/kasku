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

/**
 * F2 Riwayat — list transaksi dari Room (urut kronologis desc). Nama kategori via
 * CategoriesRepository (cache in-memory); ensureLoaded() di init memicu re-emit list
 * dengan nama benar begitu cache datang (via combine categories.version di repo).
 */
class TransactionHistoryViewModel(
    transactions: TransactionsRepository,
    private val categories: CategoriesRepository,
) : ViewModel() {

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
        ) = viewModelFactory {
            initializer { TransactionHistoryViewModel(transactions, categories) }
        }
    }
}
