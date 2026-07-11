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

class RegisterViewModel(private val repo: AuthRepository) : ViewModel() {

    var email by mutableStateOf("")
    var username by mutableStateOf("")
    var password by mutableStateOf("")
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    // Pesan sukses = tanda "terdaftar, cek email". Screen menampilkan state selesai.
    var successMessage by mutableStateOf<String?>(null)
        private set

    val canSubmit: Boolean
        get() = email.isNotBlank() && username.isNotBlank() && password.length >= 6

    fun register() {
        if (loading || !canSubmit) return
        loading = true
        error = null
        viewModelScope.launch {
            repo.register(email.trim(), username.trim(), password).fold(
                onSuccess = {
                    loading = false
                    successMessage = it
                },
                onFailure = {
                    loading = false
                    error = it.message ?: "Registrasi gagal."
                },
            )
        }
    }

    companion object {
        fun factory(repo: AuthRepository) = viewModelFactory {
            initializer { RegisterViewModel(repo) }
        }
    }
}
