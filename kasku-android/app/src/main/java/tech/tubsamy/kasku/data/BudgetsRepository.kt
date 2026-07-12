package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import tech.tubsamy.kasku.data.remote.FinanceApi

/** Anggaran read-only untuk seksi Dashboard (backend sudah hitung progress). */
data class BudgetItem(
    val id: String,
    val name: String,
    val limitIdr: Long,
    val spentIdr: Long,
    val remainingIdr: Long,
    val progressPercent: Int, // 0..∞ (bisa >100 saat lewat limit)
    val isOverBudget: Boolean,
    val alertThreshold: Int, // persen; 0 = tanpa ambang
    val categoryName: String,
)

/**
 * Pola sama dengan CategoriesRepository: TIDAK di Room — fetch online (GET /v1/budgets)
 * lalu cache in-memory sebagai StateFlow. sync-service belum meng-sync budgets, jadi
 * offline-first penuh belum bisa; gagal fetch → cache lama dipertahankan (tak throw).
 */
class BudgetsRepository(
    private val financeApi: FinanceApi,
) {
    private val _budgets = MutableStateFlow<List<BudgetItem>>(emptyList())
    val budgets: StateFlow<List<BudgetItem>> = _budgets

    /** Fetch ulang; offline/5xx → diam, nilai lama tetap tampil. */
    suspend fun refresh() {
        val loaded = runCatching {
            (financeApi.listBudgets().data ?: emptyList()).map {
                BudgetItem(
                    id = it.id,
                    name = it.name,
                    limitIdr = it.limitIdr,
                    spentIdr = it.spentIdr,
                    remainingIdr = it.remainingIdr,
                    progressPercent = it.progressPercent.toInt(),
                    isOverBudget = it.isOverBudget,
                    alertThreshold = it.alertThreshold,
                    categoryName = it.categoryName,
                )
            }
        }.getOrNull() ?: return
        _budgets.value = loaded
    }
}
