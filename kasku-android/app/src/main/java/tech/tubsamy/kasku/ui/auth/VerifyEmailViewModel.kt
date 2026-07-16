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

/**
 * Verifikasi email via token (dari link email) + kirim ulang link verifikasi via email.
 * successMessage = verifikasi berhasil (selesai). resendMessage = konfirmasi kirim ulang.
 */
class VerifyEmailViewModel(private val repo: AuthRepository) : ViewModel() {

    var token by mutableStateOf("")
    var email by mutableStateOf("")
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var successMessage by mutableStateOf<String?>(null)
        private set
    var resendMessage by mutableStateOf<String?>(null)
        private set

    val canVerify: Boolean
        get() = !loading && token.isNotBlank()

    val canResend: Boolean
        get() = !loading && email.isNotBlank()

    fun verify() {
        if (!canVerify) return
        loading = true
        error = null
        resendMessage = null
        viewModelScope.launch {
            repo.verifyEmail(token.trim()).fold(
                onSuccess = {
                    loading = false
                    successMessage = it
                },
                onFailure = {
                    loading = false
                    error = it.message ?: "Verifikasi email gagal."
                },
            )
        }
    }

    fun resend() {
        if (!canResend) return
        loading = true
        error = null
        resendMessage = null
        viewModelScope.launch {
            repo.resendVerification(email.trim()).fold(
                onSuccess = {
                    loading = false
                    resendMessage = it
                },
                onFailure = {
                    loading = false
                    error = it.message ?: "Gagal mengirim ulang verifikasi."
                },
            )
        }
    }

    companion object {
        fun factory(repo: AuthRepository) = viewModelFactory {
            initializer { VerifyEmailViewModel(repo) }
        }
    }
}
