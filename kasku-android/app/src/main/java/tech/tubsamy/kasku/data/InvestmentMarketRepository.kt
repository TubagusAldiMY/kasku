package tech.tubsamy.kasku.data

import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import tech.tubsamy.kasku.data.remote.InvestmentApi
import tech.tubsamy.kasku.data.remote.PriceApi
import tech.tubsamy.kasku.data.remote.dto.RecordUnitRequest

/** Harga live satu simbol (dari price-service). */
data class MarketPrice(val priceIdr: Double, val isFresh: Boolean)

/** Satu entri riwayat beli/jual untuk UI. */
data class InvestmentTxItem(
    val id: String,
    val transactionType: String, // BUY | SELL | ADJUST
    val quantityChange: Double,
    val pricePerUnit: Double,
    val totalValue: Double,
    val transactionDate: String, // "YYYY-MM-DD"
    val notes: String,
) {
    val isBuy: Boolean get() = transactionType == "BUY"
}

/**
 * Sisi ONLINE investasi (hybrid, pola sama BudgetsRepository/DebtsRepository): harga live +
 * riwayat + pencatatan transaksi via REST. Daftar instrumen sendiri tetap offline-first (Room).
 * Cache harga in-memory sebagai StateFlow, di-key per SIMBOL. Gagal fetch harga → diam
 * (kartu tampil tanpa live price). Pencatatan transaksi → throw bila gagal (VM tampilkan error).
 */
class InvestmentMarketRepository(
    private val priceApi: PriceApi,
    private val investmentApi: InvestmentApi,
) {
    private val _prices = MutableStateFlow<Map<String, MarketPrice>>(emptyMap())
    val prices: StateFlow<Map<String, MarketPrice>> = _prices

    /** Fetch harga satu simbol; offline/error → diam, cache lama dipertahankan. */
    suspend fun fetchPrice(symbol: String) {
        val data = runCatching { priceApi.getPrice(symbol).data }.getOrNull() ?: return
        _prices.value = _prices.value + (symbol to MarketPrice(data.priceIdr, data.isFresh))
    }

    /** Fetch semua simbol unik secara paralel (allSettled — kegagalan satu tak menggagalkan lain). */
    suspend fun fetchPrices(symbols: List<String>) = coroutineScope {
        symbols.filter { it.isNotBlank() }.distinct()
            .map { async { fetchPrice(it) } }
            .awaitAll()
    }

    /** Riwayat satu aset — di-fetch on demand (tidak di-cache). */
    suspend fun listHistory(assetId: String): List<InvestmentTxItem> =
        (investmentApi.getHistory(assetId).data ?: emptyList()).map {
            InvestmentTxItem(
                id = it.id,
                transactionType = it.transactionType,
                quantityChange = it.quantityChange,
                pricePerUnit = it.pricePerUnit,
                totalValue = it.totalValue,
                transactionDate = it.transactionDate.take(10), // RFC3339 → tanggal saja
                notes = it.notes,
            )
        }

    /** Catat BUY/SELL — server hitung ulang units & avg. Caller trigger sync utk refresh Room. */
    suspend fun recordUnit(
        assetId: String,
        transactionType: String,
        quantityChange: Double,
        pricePerUnit: Double,
        transactionDate: String,
        notes: String,
    ) {
        investmentApi.recordUnit(
            assetId,
            RecordUnitRequest(
                transactionType = transactionType,
                quantityChange = quantityChange,
                pricePerUnit = pricePerUnit,
                transactionDate = transactionDate,
                notes = notes.trim(),
            ),
        )
    }
}
