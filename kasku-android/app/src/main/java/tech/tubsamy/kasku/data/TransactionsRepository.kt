package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import tech.tubsamy.kasku.data.remote.TransactionApi
import tech.tubsamy.kasku.data.remote.dto.TransactionRequest

/**
 * Item transaksi untuk UI. type = INCOME | EXPENSE. amountIdr selalu positif; sign (prefix
 * "−") ditentukan di UI. categoryName hasil resolusi cache CategoriesRepository. accountId
 * dipakai prefill layar Edit (tak ditampilkan di daftar).
 */
data class TransactionItem(
    val id: String,
    val type: String,
    val amountIdr: Long,
    val date: String, // "YYYY-MM-DD"
    val categoryId: String?,
    val categoryName: String,
    val notes: String?,
    val accountId: String?,
)

/**
 * ONLINE (pola BudgetsRepository): fetch GET /transactions → cache in-memory StateFlow,
 * urut tanggal desc. Nama kategori diresolusi saat map lewat CategoriesRepository (dijamin
 * termuat karena refresh() memanggil ensureLoaded() dulu). Mutasi CRUD → REST lalu refresh().
 */
class TransactionsRepository(
    private val api: TransactionApi,
    private val categories: CategoriesRepository,
) {
    private val _transactions = MutableStateFlow<List<TransactionItem>>(emptyList())
    val transactions: StateFlow<List<TransactionItem>> = _transactions

    /** Ambil dari cache tanpa fetch (prefill layar edit). */
    fun find(id: String): TransactionItem? = _transactions.value.firstOrNull { it.id == id }

    /** Fetch ulang; offline/5xx → diam, nilai lama tetap tampil. */
    suspend fun refresh() {
        // Pastikan kategori termuat dulu supaya categoryName benar sejak emit pertama.
        categories.ensureLoaded()
        val loaded = runCatching {
            (api.listTransactions().data ?: emptyList())
                .map {
                    TransactionItem(
                        id = it.id,
                        type = it.transactionType,
                        amountIdr = it.amountIdr,
                        date = it.transactionDate.take(10), // trim RFC3339 → "YYYY-MM-DD"
                        categoryId = it.categoryId,
                        categoryName = categories.nameFor(it.categoryId),
                        notes = it.notes,
                        accountId = it.accountId,
                    )
                }
                .sortedByDescending { it.date }
        }.getOrNull() ?: return
        _transactions.value = loaded
    }

    suspend fun create(
        accountId: String,
        type: String,
        amountIdr: Long,
        categoryId: String?,
        date: String,
        notes: String,
    ) {
        api.createTransaction(
            TransactionRequest(
                accountId = accountId,
                categoryId = categoryId,
                transactionType = type,
                amountIdr = amountIdr,
                transactionDate = date,
                notes = notes,
            ),
        )
        refresh()
    }

    suspend fun update(
        id: String,
        accountId: String,
        type: String,
        amountIdr: Long,
        categoryId: String?,
        date: String,
        notes: String,
    ) {
        api.updateTransaction(
            id,
            TransactionRequest(
                accountId = accountId,
                categoryId = categoryId,
                transactionType = type,
                amountIdr = amountIdr,
                transactionDate = date,
                notes = notes,
            ),
        )
        refresh()
    }

    suspend fun delete(id: String) {
        api.deleteTransaction(id)
        refresh()
    }
}
