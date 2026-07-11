package tech.tubsamy.kasku.ui.dashboard

import android.content.Context
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.CategoriesRepository
import tech.tubsamy.kasku.data.DashboardSummary
import tech.tubsamy.kasku.data.TransactionsRepository
import tech.tubsamy.kasku.data.sync.SyncWorker
import tech.tubsamy.kasku.ui.formatIdr
import java.time.LocalDate
import java.time.YearMonth

/**
 * F1 Dashboard — dihitung LOKAL dari Room (offline-first, tanpa endpoint baru).
 * combine(akun aktif, transaksi bulan berjalan) → DashboardSummary.compute (pure).
 *
 * `today` disuntik supaya rentang bulan testable/KMP-ready (default LocalDate.now).
 */
class DashboardViewModel(
    accounts: AccountsRepository,
    transactions: TransactionsRepository,
    private val categories: CategoriesRepository,
    private val appContext: Context,
    today: () -> LocalDate = { LocalDate.now() },
) : ViewModel() {

    val summary: StateFlow<DashboardSummary> = run {
        val month = YearMonth.from(today())
        val from = month.atDay(1).toString()   // "YYYY-MM-DD"
        val to = month.atEndOfMonth().toString()
        combine(
            accounts.observeAccounts(),
            transactions.observeMonth(from, to),
        ) { accs, txs ->
            DashboardSummary.compute(accs, txs) { formatIdr(it) }
        }.stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = DashboardSummary(0, 0, 0, 0, emptyList()),
        )
    }

    init {
        // Cache kategori untuk insight "pengeluaran terbesar" + segarkan Room.
        viewModelScope.launch { categories.ensureLoaded() }
        SyncWorker.enqueueOnce(appContext)
    }

    companion object {
        fun factory(
            accounts: AccountsRepository,
            transactions: TransactionsRepository,
            categories: CategoriesRepository,
            appContext: Context,
        ) = viewModelFactory {
            initializer {
                DashboardViewModel(
                    accounts,
                    transactions,
                    categories,
                    appContext.applicationContext,
                )
            }
        }
    }
}
