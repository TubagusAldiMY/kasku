package tech.tubsamy.kasku.ui.profile

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
 * Form ganti password. Bearer disisipkan otomatis oleh interceptor jaringan, jadi
 * hanya butuh current+new. Validasi kekuatan penuh (huruf besar/kecil/angka)
 * ditegakkan backend — client cek panjang >= 8 dan konfirmasi cocok.
 */
class ChangePasswordViewModel(
    private val auth: AuthRepository,
) : ViewModel() {

    var currentPassword by mutableStateOf("")
    var newPassword by mutableStateOf("")
    var confirmPassword by mutableStateOf("")
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    val canSave: Boolean
        get() = !saving &&
            currentPassword.isNotBlank() &&
            newPassword.length >= 8 &&
            newPassword == confirmPassword

    fun save(onDone: () -> Unit) {
        if (!canSave) return
        saving = true
        error = null
        viewModelScope.launch {
            auth.changePassword(currentPassword, newPassword)
                .onSuccess {
                    saving = false
                    onDone()
                }
                .onFailure {
                    saving = false
                    error = it.message ?: "Gagal mengubah password."
                }
        }
    }

    companion object {
        fun factory(auth: AuthRepository) = viewModelFactory {
            initializer { ChangePasswordViewModel(auth) }
        }
    }
}
