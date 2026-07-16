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

class ResetPasswordViewModel(private val repo: AuthRepository) : ViewModel() {

    var token by mutableStateOf("")
    var newPassword by mutableStateOf("")
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var successMessage by mutableStateOf<String?>(null)
        private set

    val canSubmit: Boolean
        get() = !loading && token.isNotBlank() && newPassword.length >= 6

    fun submit() {
        if (!canSubmit) return
        loading = true
        error = null
        viewModelScope.launch {
            repo.resetPassword(token.trim(), newPassword).fold(
                onSuccess = {
                    loading = false
                    successMessage = it
                },
                onFailure = {
                    loading = false
                    error = it.message ?: "Reset password gagal."
                },
            )
        }
    }

    companion object {
        fun factory(repo: AuthRepository) = viewModelFactory {
            initializer { ResetPasswordViewModel(repo) }
        }
    }
}
