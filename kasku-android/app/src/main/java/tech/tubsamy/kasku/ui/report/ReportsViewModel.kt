package tech.tubsamy.kasku.ui.report

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.ReportsRepository
import tech.tubsamy.kasku.data.remote.dto.ReportDto

/** Layar laporan keuangan (read-only). Muat saat dibuka, tombol coba lagi saat gagal. */
class ReportsViewModel(
    private val repo: ReportsRepository,
) : ViewModel() {

    var report by mutableStateOf<ReportDto?>(null)
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
                report = repo.load()
            } catch (e: Exception) {
                error = "Gagal memuat laporan. Periksa koneksi."
            } finally {
                loading = false
            }
        }
    }

    companion object {
        fun factory(repo: ReportsRepository) = viewModelFactory {
            initializer { ReportsViewModel(repo) }
        }
    }
}
