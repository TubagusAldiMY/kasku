package tech.tubsamy.kasku.ui.dashboard

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.BudgetItem
import tech.tubsamy.kasku.data.BudgetsRepository
import tech.tubsamy.kasku.data.CategoriesRepository
import tech.tubsamy.kasku.data.DashboardCharts
import tech.tubsamy.kasku.data.DashboardSummary
import tech.tubsamy.kasku.data.TransactionsRepository
import tech.tubsamy.kasku.ui.formatIdr
import java.time.LocalDate
import java.time.YearMonth

/**
 * F1 Dashboard — dihitung LOKAL dari cache repo ONLINE (accounts + transactions), tanpa
 * endpoint agregasi baru. combine(akun, transaksi bulan berjalan) → DashboardSummary.compute
 * (pure). `today` disuntik supaya rentang bulan testable (default LocalDate.now).
 */
class DashboardViewModel(
    private val accounts: AccountsRepository,
    private val transactions: TransactionsRepository,
    private val categories: CategoriesRepository,
    private val budgetsRepo: BudgetsRepository,
    today: () -> LocalDate = { LocalDate.now() },
) : ViewModel() {

    private val month = YearMonth.from(today())
    private val monthFrom = month.atDay(1).toString()   // "YYYY-MM-DD"
    private val monthTo = month.atEndOfMonth().toString()

    /** "Juli 2026" — konteks eyebrow hero. */
    val monthLabel: String =
        month.month.getDisplayName(java.time.format.TextStyle.FULL, java.util.Locale.forLanguageTag("id")) +
            " " + month.year

    val summary: StateFlow<DashboardSummary> =
        combine(
            accounts.accounts,
            transactions.transactions,
        ) { accs, txs ->
            val monthTxs = txs.filter { it.date in monthFrom..monthTo }
            DashboardSummary.compute(accs, monthTxs) { formatIdr(it) }
        }.stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = DashboardSummary(0, 0, 0, 0, emptyList()),
        )

    /**
     * Data grafik dari SELURUH riwayat — trend butuh beberapa bulan. Dihitung LOKAL/pure;
     * breakdown difilter ulang ke bulan berjalan di sini.
     */
    val charts: StateFlow<DashboardCharts> =
        transactions.transactions.map { txs ->
            DashboardCharts(
                trend = DashboardCharts.trend(txs, month.toString(), months = TREND_MONTHS),
                expenseByCategory = DashboardCharts.expenseByCategory(
                    txs.filter { it.date in monthFrom..monthTo },
                ),
            )
        }.stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = DashboardCharts.EMPTY,
        )

    /** 4 transaksi terakhir — seksi "Aktivitas terakhir". */
    val recent: StateFlow<List<tech.tubsamy.kasku.data.TransactionItem>> =
        transactions.transactions.map { txs ->
            txs.sortedByDescending { it.date }.take(RECENT_COUNT)
        }.stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = emptyList(),
        )

    /** Anggaran + progress dari backend (online, cache in-memory di repo). */
    val budgets: StateFlow<List<BudgetItem>> = budgetsRepo.budgets

    init {
        viewModelScope.launch { categories.ensureLoaded() }
        viewModelScope.launch { accounts.refresh() }
        viewModelScope.launch { transactions.refresh() }
        viewModelScope.launch { budgetsRepo.refresh() }
    }

    companion object {
        private const val TREND_MONTHS = 6
        private const val RECENT_COUNT = 4

        fun factory(
            accounts: AccountsRepository,
            transactions: TransactionsRepository,
            categories: CategoriesRepository,
            budgets: BudgetsRepository,
        ) = viewModelFactory {
            initializer { DashboardViewModel(accounts, transactions, categories, budgets) }
        }
    }
}
