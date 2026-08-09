package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import tech.tubsamy.kasku.data.remote.InvestmentApi
import tech.tubsamy.kasku.data.remote.dto.CreateAssetRequest
import tech.tubsamy.kasku.data.remote.dto.UpdateAssetRequest

/** Item investasi untuk UI. */
data class InvestmentItem(
    val id: String,
    val name: String,
    val assetType: String,
    val symbol: String?,
    val units: Double,
    val avgBuyPriceIdr: Long,
) {
    /** Total nilai buku = units × harga beli rata-rata (rupiah bulat). */
    val bookValueIdr: Long get() = (units * avgBuyPriceIdr).toLong()
}

/**
 * ONLINE (pola BudgetsRepository): fetch GET /investments → cache in-memory StateFlow.
 * Web memetakan quantity→units & avg_buy_price→avgBuyPriceIdr; sama di sini. units & avg
 * hanya bisa diubah lewat pencatatan unit (BUY/SELL), bukan update instrumen.
 */
class InvestmentsRepository(
    private val api: InvestmentApi,
) {
    private val _investments = MutableStateFlow<List<InvestmentItem>>(emptyList())
    val investments: StateFlow<List<InvestmentItem>> = _investments

    /** Ambil dari cache tanpa fetch (prefill layar edit). */
    fun find(id: String): InvestmentItem? = _investments.value.firstOrNull { it.id == id }

    /** Fetch ulang; offline/5xx → diam, nilai lama tetap tampil. */
    suspend fun refresh() {
        val loaded = runCatching {
            (api.listInvestments().data ?: emptyList()).map {
                InvestmentItem(
                    id = it.id,
                    name = it.name,
                    assetType = it.assetType,
                    symbol = it.symbol.ifBlank { null },
                    units = it.quantity,
                    avgBuyPriceIdr = it.avgBuyPrice.toLong(),
                )
            }
        }.getOrNull() ?: return
        _investments.value = loaded
    }

    suspend fun create(
        name: String,
        assetType: String,
        units: Double,
        avgBuyPriceIdr: Long,
        symbol: String,
    ) {
        api.createInvestment(
            CreateAssetRequest(
                name = name.trim(),
                assetType = assetType,
                symbol = symbol.trim(),
                quantity = units,
                avgBuyPrice = avgBuyPriceIdr.toDouble(),
            ),
        )
        refresh()
    }

    /** Hanya name/asset_type/symbol yang dapat diubah — units & avg via pencatatan unit. */
    suspend fun update(id: String, name: String, assetType: String, symbol: String) {
        api.updateInvestment(
            id,
            UpdateAssetRequest(name = name.trim(), assetType = assetType, symbol = symbol.trim()),
        )
        refresh()
    }

    suspend fun delete(id: String) {
        api.deleteInvestment(id)
        refresh()
    }
}
