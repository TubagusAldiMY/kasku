package tech.tubsamy.kasku.ui.profile

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.ProfileRepository
import tech.tubsamy.kasku.data.remote.dto.ProfileDto

/** Layar profil (read-only). Menampilkan identitas + tier langganan. */
class ProfileViewModel(
    private val repo: ProfileRepository,
) : ViewModel() {

    var profile by mutableStateOf<ProfileDto?>(null)
        private set
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    init {
        load()
    }

    fun load() {
        loading = true
        error = null
        viewModelScope.launch {
            try {
                profile = repo.load()
            } catch (e: Exception) {
                error = "Gagal memuat profil. Periksa koneksi."
            } finally {
                loading = false
            }
        }
    }

    companion object {
        fun factory(repo: ProfileRepository) = viewModelFactory {
            initializer { ProfileViewModel(repo) }
        }
    }
}
