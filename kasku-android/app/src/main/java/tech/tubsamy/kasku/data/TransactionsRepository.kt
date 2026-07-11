package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.combine
import tech.tubsamy.kasku.data.local.KasKuDatabase
import tech.tubsamy.kasku.data.local.TransactionEntity

/**
 * Item transaksi untuk UI. type = INCOME | EXPENSE. amountIdr selalu positif;
 * sign (prefix "−") ditentukan di UI. categoryName hasil resolusi cache in-memory.
 */
data class TransactionItem(
    val id: String,
    val type: String,
    val amountIdr: Long,
    val date: String, // "YYYY-MM-DD"
    val categoryId: String?,
    val categoryName: String,
    val notes: String?,
)

/**
 * Infra bersama F1 (Dashboard bulan berjalan) + F2 (Riwayat). Sumber kebenaran = Room.
 *
 * Nama kategori di-resolve saat map lewat CategoriesRepository.nameFor (cache in-memory).
 * Di-combine dengan categories.version → list re-emit dengan nama benar begitu cache datang
 * (hindari "Umum" nyangkut saat emit pertama mendahului fetch).
 */
class TransactionsRepository(
    db: KasKuDatabase,
    private val categories: CategoriesRepository,
) {
    private val dao = db.transactionDao()

    /** F2 Riwayat: semua transaksi, urut kronologis desc. */
    fun observeAll(): Flow<List<TransactionItem>> =
        dao.observe().withCategoryNames()

    /** F1 Dashboard: bulan berjalan; from/to "YYYY-MM-DD" di-supply VM via clock. */
    fun observeMonth(from: String, to: String): Flow<List<TransactionItem>> =
        dao.observeByDateRange(from, to).withCategoryNames()

    private fun Flow<List<TransactionEntity>>.withCategoryNames(): Flow<List<TransactionItem>> =
        combine(categories.version) { rows, _ -> rows.map { it.toItem() } }

    private fun TransactionEntity.toItem() = TransactionItem(
        id = id,
        type = transaction_type,
        amountIdr = amount_idr,
        date = transaction_date,
        categoryId = category_id,
        categoryName = categories.nameFor(category_id),
        notes = notes,
    )
}
