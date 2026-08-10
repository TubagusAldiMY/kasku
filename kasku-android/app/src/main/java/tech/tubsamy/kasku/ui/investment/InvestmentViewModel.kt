package tech.tubsamy.kasku.ui.investment

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

/**
 * ONLINE: daftar instrumen dari cache InvestmentsRepository (GET /investments) + harga live
 * (price-service). Masuk layar → refresh(); setiap kali kumpulan simbol berubah → fetch harga.
 */
class InvestmentViewModel(
    private val repo: InvestmentsRepository,
    private val market: InvestmentMarketRepository,
) : ViewModel() {

    val investments: StateFlow<List<InvestmentItem>> = repo.investments

    /** Cache harga live, di-key per simbol. */
    val prices: StateFlow<Map<String, MarketPrice>> =
        market.prices.stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = emptyMap(),
        )

    init {
        viewModelScope.launch { repo.refresh() }
        // Fetch harga tiap kali kumpulan simbol berubah (bukan tiap perubahan lain).
        viewModelScope.launch {
            investments
                .map { list -> list.mapNotNull { it.symbol?.takeIf { s -> s.isNotBlank() } }.distinct().sorted() }
                .distinctUntilChanged()
                .collect { symbols -> if (symbols.isNotEmpty()) market.fetchPrices(symbols) }
        }
    }

    /** Segarkan: tarik ulang daftar + refetch harga live untuk simbol saat ini. */
    fun refresh() {
        viewModelScope.launch {
            repo.refresh()
            market.fetchPrices(investments.value.mapNotNull { it.symbol })
        }
    }

    companion object {
        fun factory(
            repo: InvestmentsRepository,
            market: InvestmentMarketRepository,
        ) = viewModelFactory {
            initializer { InvestmentViewModel(repo, market) }
        }
    }
}
