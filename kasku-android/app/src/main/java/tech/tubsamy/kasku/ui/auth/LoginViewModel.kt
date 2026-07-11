package tech.tubsamy.kasku.ui.auth

import android.content.Context
import android.util.Log
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.credentials.exceptions.GetCredentialCancellationException
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.AuthRepository
import tech.tubsamy.kasku.data.remote.GoogleAuthClient

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

    /** Google Sign-In native: ambil ID token (Credential Manager) → POST /auth/google. */
    fun googleSignIn(context: Context, onSuccess: () -> Unit) {
        if (loading) return
        loading = true
        error = null
        viewModelScope.launch {
            try {
                val idToken = GoogleAuthClient.getIdToken(context)
                repo.googleLogin(idToken).fold(
                    onSuccess = {
                        loading = false
                        onSuccess()
                    },
                    onFailure = {
                        loading = false
                        error = it.message ?: "Login Google gagal."
                    },
                )
            } catch (e: GetCredentialCancellationException) {
                loading = false // user membatalkan — diamkan
            } catch (e: Exception) {
                loading = false
                // Log penuh untuk diagnosa (no silent failures); pesan UI tetap ramah.
                Log.e("KasKuAuth", "google sign-in gagal", e)
                error = "Login Google gagal. Coba lagi."
            }
        }
    }

    companion object {
        fun factory(repo: AuthRepository) = viewModelFactory {
            initializer { LoginViewModel(repo) }
        }
    }
}
