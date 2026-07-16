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

class ForgotPasswordViewModel(private val repo: AuthRepository) : ViewModel() {

    var email by mutableStateOf("")
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    // Backend selalu balas generik (anti-enumeration) — success = tampilkan konfirmasi.
    var successMessage by mutableStateOf<String?>(null)
        private set

    val canSubmit: Boolean
        get() = !loading && email.isNotBlank()

    fun submit() {
        if (!canSubmit) return
        loading = true
        error = null
        viewModelScope.launch {
            repo.forgotPassword(email.trim()).fold(
                onSuccess = {
                    loading = false
                    successMessage = it
                },
                onFailure = {
                    loading = false
                    error = it.message ?: "Gagal mengirim permintaan reset."
                },
            )
        }
    }

    companion object {
        fun factory(repo: AuthRepository) = viewModelFactory {
            initializer { ForgotPasswordViewModel(repo) }
        }
    }
}
