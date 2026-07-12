package tech.tubsamy.kasku.ui.dashboard

import android.content.Context
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
import tech.tubsamy.kasku.data.CategoriesRepository
import tech.tubsamy.kasku.data.DashboardCharts
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

    private val month = YearMonth.from(today())
    private val monthFrom = month.atDay(1).toString()   // "YYYY-MM-DD"
    private val monthTo = month.atEndOfMonth().toString()

    /** "Juli 2026" — konteks eyebrow hero (desain ReDesign/). */
    val monthLabel: String =
        month.month.getDisplayName(java.time.format.TextStyle.FULL, java.util.Locale.forLanguageTag("id")) +
            " " + month.year

    val summary: StateFlow<DashboardSummary> =
        combine(
            accounts.observeAccounts(),
            transactions.observeMonth(monthFrom, monthTo),
        ) { accs, txs ->
            DashboardSummary.compute(accs, txs) { formatIdr(it) }
        }.stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = DashboardSummary(0, 0, 0, 0, emptyList()),
        )

    /**
     * Data grafik dari SELURUH riwayat (observeAll) — trend butuh beberapa bulan, tak cukup
     * observeMonth. Dihitung LOKAL/pure; breakdown difilter ulang ke bulan berjalan di sini.
     */
    val charts: StateFlow<DashboardCharts> =
        transactions.observeAll().map { txs ->
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

    /** 4 transaksi terakhir — seksi "Aktivitas terakhir" (desain ReDesign/). */
    val recent: StateFlow<List<tech.tubsamy.kasku.data.TransactionItem>> =
        transactions.observeAll().map { txs ->
            txs.sortedByDescending { it.date }.take(RECENT_COUNT)
        }.stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = emptyList(),
        )

    init {
        // Cache kategori untuk insight "pengeluaran terbesar" + segarkan Room.
        viewModelScope.launch { categories.ensureLoaded() }
        SyncWorker.enqueueOnce(appContext)
    }

    companion object {
        private const val TREND_MONTHS = 6
        private const val RECENT_COUNT = 4

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
