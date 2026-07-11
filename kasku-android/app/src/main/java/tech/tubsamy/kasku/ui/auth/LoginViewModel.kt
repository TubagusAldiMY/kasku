package tech.tubsamy.kasku.ui.auth

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.AuthRepository

class LoginViewModel(private val repo: AuthRepository) : ViewModel() {

    var email by mutableStateOf("")
    var password by mutableStateOf("")
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    fun login(onSuccess: () -> Unit) {
        if (loading) return
        loading = true
        error = null
        viewModelScope.launch {
            repo.login(email.trim(), password).fold(
                onSuccess = {
                    loading = false
                    onSuccess()
                },
                onFailure = {
                    loading = false
                    error = it.message ?: "Login gagal."
                },
            )
        }
    }

    companion object {
        fun factory(repo: AuthRepository) = viewModelFactory {
            initializer { LoginViewModel(repo) }
        }
    }
}
