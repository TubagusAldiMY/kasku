package tech.tubsamy.kasku.ui.investment

import android.content.Context
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.InvestmentItem
import tech.tubsamy.kasku.data.InvestmentMarketRepository
import tech.tubsamy.kasku.data.InvestmentsRepository
import tech.tubsamy.kasku.data.MarketPrice
import tech.tubsamy.kasku.data.sync.SyncWorker

/**
 * OFFLINE-FIRST daftar instrumen (Room, diisi sync) + ONLINE harga live (price-service).
 * Masuk layar → picu sync sekali; setiap kali daftar simbol berubah → fetch harga live.
 */
class InvestmentViewModel(
    private val repo: InvestmentsRepository,
    private val market: InvestmentMarketRepository,
    private val appContext: Context,
) : ViewModel() {

    val investments: StateFlow<List<InvestmentItem>> =
        repo.observeInvestments().stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = emptyList(),
        )

    /** Cache harga live, di-key per simbol. */
    val prices: StateFlow<Map<String, MarketPrice>> =
        market.prices.stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = emptyMap(),
        )

    init {
        SyncWorker.enqueueOnce(appContext)
        // Fetch harga tiap kali kumpulan simbol berubah (bukan tiap perubahan Room lain).
        viewModelScope.launch {
            investments
                .map { list -> list.mapNotNull { it.symbol?.takeIf { s -> s.isNotBlank() } }.distinct().sorted() }
                .distinctUntilChanged()
                .collect { symbols -> if (symbols.isNotEmpty()) market.fetchPrices(symbols) }
        }
    }

    /** Segarkan: sync ulang daftar + refetch harga live untuk simbol saat ini. */
    fun refresh() {
        SyncWorker.enqueueOnce(appContext)
        viewModelScope.launch {
            market.fetchPrices(investments.value.mapNotNull { it.symbol })
        }
    }

    companion object {
        fun factory(
            repo: InvestmentsRepository,
            market: InvestmentMarketRepository,
            appContext: Context,
        ) = viewModelFactory {
            initializer { InvestmentViewModel(repo, market, appContext.applicationContext) }
        }
    }
}
