package tech.tubsamy.kasku.ui.budget

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.BudgetsRepository
import tech.tubsamy.kasku.data.CategoriesRepository

/**
 * Form tambah/edit anggaran. editId != null → prefill dari cache repo (navigasi selalu
 * dari daftar yang baru fetch, jadi cache hangat). Periode/threshold tidak diedit UI —
 * backend default MONTHLY/80; saat edit, alertThreshold lama diteruskan apa adanya.
 */
class AddBudgetViewModel(
    private val budgets: BudgetsRepository,
    val categories: CategoriesRepository,
    private val editId: String? = null,
) : ViewModel() {

    val isEdit: Boolean get() = editId != null

    var name by mutableStateOf("")
    var limit by mutableStateOf("") // digit string, rupiah
    var categoryId by mutableStateOf<String?>(null)
    var dailyLimitEnabled by mutableStateOf(false)
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    // alertThreshold lama untuk passthrough PUT (0 saat create — backend default 80).
    private var alertThreshold = 0

    val canSave: Boolean
        get() = !saving && name.isNotBlank() && (limit.toLongOrNull() ?: 0L) > 0L

    init {
        viewModelScope.launch { categories.ensureLoaded() }
        editId?.let { id ->
            budgets.budgets.value.firstOrNull { it.id == id }?.let { b ->
                name = b.name
                limit = b.limitIdr.toString()
                categoryId = b.categoryId
                dailyLimitEnabled = b.dailyLimitEnabled
                alertThreshold = b.alertThreshold
            } ?: run { error = "Anggaran tidak ditemukan." }
        }
    }

    fun save(onSaved: () -> Unit) {
        val limitIdr = limit.toLongOrNull() ?: 0L
        if (!canSave) return
        saving = true
        error = null
        viewModelScope.launch {
            try {
                if (editId != null) {
                    budgets.update(editId, name, limitIdr, categoryId, alertThreshold, dailyLimitEnabled)
                } else {
                    budgets.create(name, limitIdr, categoryId, dailyLimitEnabled)
                }
                saving = false
                onSaved()
            } catch (e: Exception) {
                saving = false
                error = "Gagal menyimpan anggaran. Periksa koneksi."
            }
        }
    }

    companion object {
        fun factory(
            budgets: BudgetsRepository,
            categories: CategoriesRepository,
            editId: String? = null,
        ) = viewModelFactory {
            initializer { AddBudgetViewModel(budgets, categories, editId) }
        }
    }
}
