package tech.tubsamy.kasku.ui.home

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.remote.dto.AccountDto

class HomeViewModel(private val repo: AccountsRepository) : ViewModel() {

    var loading by mutableStateOf(false)
        private set
    var accounts by mutableStateOf<List<AccountDto>>(emptyList())
        private set
    var error by mutableStateOf<String?>(null)
        private set

    init {
        load()
    }

    fun load() {
        if (loading) return
        loading = true
        error = null
        viewModelScope.launch {
            repo.listAccounts().fold(
                onSuccess = {
                    accounts = it
                    loading = false
                },
                onFailure = {
                    error = it.message ?: "Gagal memuat akun."
                    loading = false
                },
            )
        }
    }

    companion object {
        fun factory(repo: AccountsRepository) = viewModelFactory {
            initializer { HomeViewModel(repo) }
        }
    }
}
