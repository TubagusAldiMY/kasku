package tech.tubsamy.kasku.ui.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.AccountItem
import tech.tubsamy.kasku.data.AccountsRepository

/**
 * ONLINE: akun dibaca dari cache StateFlow AccountsRepository (GET /accounts).
 * Masuk Home → refresh() sekali. Mutasi (hapus) langsung REST lalu repo refresh cache.
 */
class HomeViewModel(
    private val repo: AccountsRepository,
) : ViewModel() {

    val accounts: StateFlow<List<AccountItem>> = repo.accounts

    init {
        refresh()
    }

    /** Tarik ulang dari server. */
    fun refresh() {
        viewModelScope.launch { repo.refresh() }
    }

    /** Hapus akun via REST, lalu cache disegarkan oleh repo. */
    fun deleteAccount(id: String) {
        viewModelScope.launch { repo.delete(id) }
    }

    companion object {
        fun factory(repo: AccountsRepository) = viewModelFactory {
            initializer { HomeViewModel(repo) }
        }
    }
}
