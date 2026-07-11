package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import tech.tubsamy.kasku.data.local.KasKuDatabase

/** Item investasi untuk UI (dari Room, hasil sinkronisasi). */
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
 * OFFLINE-FIRST (M2): sumber kebenaran tampilan = Room, diisi oleh SyncEngine.
 * Sama seperti AccountsRepository — tidak memanggil REST, sync yang mengisi tabel.
 */
class InvestmentsRepository(
    db: KasKuDatabase,
) {
    private val investmentDao = db.investmentDao()

    /** Investasi aktif, reaktif. Kosong sampai sync pertama mengisi Room. */
    fun observeInvestments(): Flow<List<InvestmentItem>> =
        investmentDao.observeActive().map { rows ->
            rows.map {
                InvestmentItem(
                    id = it.id,
                    name = it.name,
                    assetType = it.asset_type,
                    symbol = it.symbol,
                    units = it.units,
                    avgBuyPriceIdr = it.avg_buy_price_idr,
                )
            }
        }
}
