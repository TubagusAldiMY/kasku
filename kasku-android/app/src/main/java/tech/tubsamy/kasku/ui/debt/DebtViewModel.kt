package tech.tubsamy.kasku.ui.debt

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.DebtItem
import tech.tubsamy.kasku.data.DebtsRepository

/** Daftar hutang & piutang: list + hapus. Form tambah/edit di AddDebtViewModel. */
class DebtViewModel(
    private val repo: DebtsRepository,
) : ViewModel() {

    val debts: StateFlow<List<DebtItem>> = repo.debts

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
                error = "Gagal menghapus catatan. Periksa koneksi."
            }
        }
    }

    companion object {
        fun factory(repo: DebtsRepository) = viewModelFactory {
            initializer { DebtViewModel(repo) }
        }
    }
}
