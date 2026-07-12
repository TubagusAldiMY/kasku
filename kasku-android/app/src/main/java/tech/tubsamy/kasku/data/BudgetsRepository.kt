package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import tech.tubsamy.kasku.data.remote.FinanceApi
import tech.tubsamy.kasku.data.remote.dto.CreateBudgetRequest
import tech.tubsamy.kasku.data.remote.dto.UpdateBudgetRequest

/** Anggaran untuk UI (backend sudah hitung progress). */
data class BudgetItem(
    val id: String,
    val name: String,
    val limitIdr: Long,
    val spentIdr: Long,
    val remainingIdr: Long,
    val progressPercent: Int, // 0..∞ (bisa >100 saat lewat limit)
    val isOverBudget: Boolean,
    val alertThreshold: Int, // persen; 0 = tanpa ambang
    val categoryId: String?,
    val categoryName: String,
    val dailyLimitEnabled: Boolean,
    // Terisi hanya saat dailyLimitEnabled = true.
    val dailyAllowanceTodayIdr: Long? = null,
    val spentTodayIdr: Long? = null,
    val dailyRemainingIdr: Long? = null,
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
                    categoryId = it.categoryId,
                    categoryName = it.categoryName,
                    dailyLimitEnabled = it.dailyLimitEnabled,
                    dailyAllowanceTodayIdr = it.dailyAllowanceTodayIdr,
                    spentTodayIdr = it.spentTodayIdr,
                    dailyRemainingIdr = it.dailyRemainingIdr,
                )
            }
        }.getOrNull() ?: return
        _budgets.value = loaded
    }

    // Mutasi CRUD — throw bila gagal (VM yang menampilkan error), sukses → refresh cache.

    suspend fun create(name: String, limitIdr: Long, categoryId: String?, dailyLimitEnabled: Boolean) {
        financeApi.createBudget(
            CreateBudgetRequest(
                name = name.trim(),
                limitIdr = limitIdr,
                categoryId = categoryId,
                dailyLimitEnabled = dailyLimitEnabled,
            ),
        )
        refresh()
    }

    suspend fun update(
        id: String,
        name: String,
        limitIdr: Long,
        categoryId: String?,
        alertThreshold: Int,
        dailyLimitEnabled: Boolean,
    ) {
        financeApi.updateBudget(
            id,
            UpdateBudgetRequest(
                name = name.trim(),
                limitIdr = limitIdr,
                categoryId = categoryId,
                alertThreshold = alertThreshold,
                dailyLimitEnabled = dailyLimitEnabled,
            ),
        )
        refresh()
    }

    suspend fun delete(id: String) {
        financeApi.deleteBudget(id)
        refresh()
    }
}
