package tech.tubsamy.kasku.ui.investment

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.InvestmentItem
import tech.tubsamy.kasku.data.InvestmentMarketRepository
import tech.tubsamy.kasku.data.InvestmentTxItem
import tech.tubsamy.kasku.data.InvestmentsRepository
import tech.tubsamy.kasku.data.MarketPrice
import java.time.LocalDate

/**
 * Rincian satu instrumen: harga live + riwayat beli/jual (GET history) + form catat BUY/SELL
 * (POST units) → repo.refresh() untuk tarik ulang units/avg yang sudah dihitung server. Aset
 * dibaca reaktif dari cache InvestmentsRepository. Edit/Hapus lewat entry point terpisah.
 */
class InvestmentDetailViewModel(
    private val repo: InvestmentsRepository,
    private val market: InvestmentMarketRepository,
    private val assetId: String,
) : ViewModel() {

    var asset by mutableStateOf<InvestmentItem?>(null)
        private set
    var price by mutableStateOf<MarketPrice?>(null)
        private set
    var history by mutableStateOf<List<InvestmentTxItem>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set

    // Form catat transaksi.
    var txType by mutableStateOf("BUY") // BUY | SELL
    var quantity by mutableStateOf("")  // desimal
    var totalValue by mutableStateOf("") // rupiah bulat — price_per_unit dihitung dari ini
    var txDate by mutableStateOf(LocalDate.now().toString()) // YYYY-MM-DD
    var notes by mutableStateOf("")
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    private val quantityValue: Double? get() = quantity.trim().toDoubleOrNull()
    private val totalValueValue: Long? get() = totalValue.filter { it.isDigit() }.toLongOrNull()

    /** Preview harga per unit dari nilai total / jumlah (0 bila belum valid). */
    val pricePerUnitPreview: Long
        get() {
            val q = quantityValue ?: return 0
            val t = totalValueValue ?: return 0
            return if (q > 0) (t / q).toLong() else 0
        }

    val canRecord: Boolean
        get() = !saving && (quantityValue ?: 0.0) > 0.0 && (totalValueValue ?: 0) > 0

    init {
        viewModelScope.launch {
            repo.refresh()
            repo.investments.collect { list ->
                val found = list.firstOrNull { it.id == assetId }
                asset = found
                if (found?.symbol?.isNotBlank() == true && price == null) loadPrice(found.symbol)
            }
        }
        loadHistory()
    }

    private fun loadPrice(symbol: String) {
        viewModelScope.launch {
            market.fetchPrice(symbol)
            price = market.prices.value[symbol]
        }
    }

    private fun loadHistory() {
        viewModelScope.launch {
            loading = true
            history = runCatching { market.listHistory(assetId) }.getOrDefault(history)
            loading = false
        }
    }

    fun record() {
        val q = quantityValue
        val total = totalValueValue
        if (!canRecord || q == null || total == null || q <= 0.0) {
            error = "Isi jumlah unit dan nilai transaksi."
            return
        }
        saving = true
        error = null
        viewModelScope.launch {
            try {
                market.recordUnit(
                    assetId = assetId,
                    transactionType = txType,
                    quantityChange = q,
                    pricePerUnit = total.toDouble() / q,
                    transactionDate = txDate,
                    notes = notes,
                )
                // Server sudah hitung ulang units & avg → tarik ulang daftar.
                repo.refresh()
                quantity = ""
                totalValue = ""
                notes = ""
                txType = "BUY"
                history = market.listHistory(assetId)
                saving = false
            } catch (e: Exception) {
                saving = false
                error = "Gagal menyimpan transaksi. Periksa koneksi atau jumlah melebihi kepemilikan."
            }
        }
    }

    fun delete(onDone: () -> Unit) {
        viewModelScope.launch {
            repo.delete(assetId)
            onDone()
        }
    }

    companion object {
        fun factory(
            repo: InvestmentsRepository,
            market: InvestmentMarketRepository,
            assetId: String,
        ) = viewModelFactory {
            initializer { InvestmentDetailViewModel(repo, market, assetId) }
        }
    }
}
