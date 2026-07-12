package tech.tubsamy.kasku.ui.budget

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.BudgetItem
import tech.tubsamy.kasku.data.BudgetsRepository

/** Layar kelola anggaran: daftar + hapus. Form tambah/edit di AddBudgetViewModel. */
class BudgetViewModel(
    private val repo: BudgetsRepository,
) : ViewModel() {

    val budgets: StateFlow<List<BudgetItem>> = repo.budgets

    var error by mutableStateOf<String?>(null)
        private set

    init {
        viewModelScope.launch { repo.refresh() }
    }

    fun refresh() {
        viewModelScope.launch { repo.refresh() }
    }

    fun delete(id: String) {
        viewModelScope.launch {
            try {
                repo.delete(id)
                error = null
            } catch (e: Exception) {
                error = "Gagal menghapus anggaran. Periksa koneksi."
            }
        }
    }

    companion object {
        fun factory(repo: BudgetsRepository) = viewModelFactory {
            initializer { BudgetViewModel(repo) }
        }
    }
}
