package tech.tubsamy.kasku.ui.category

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.CategoriesRepository
import tech.tubsamy.kasku.data.CategoryItem

/**
 * Layar kelola kategori: daftar + form tambah (nama + tipe INCOME/EXPENSE/BOTH).
 *
 * Daftar reaktif ke categories.version — naik tiap fetch/create sukses → list re-emit tanpa
 * observer manual. Tipe valid divalidasi di sini (canSave) sebelum kirim; backend juga
 * meng-enforce oneof, jadi Zero Trust: dua lapis.
 */
class CategoryViewModel(
    private val categories: CategoriesRepository,
) : ViewModel() {

    val items: StateFlow<List<CategoryItem>> =
        categories.version
            .map { categories.all() }
            .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    var name by mutableStateOf("")
    var type by mutableStateOf("EXPENSE") // INCOME | EXPENSE | BOTH
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    private val validType = setOf("INCOME", "EXPENSE", "BOTH")

    val canSave: Boolean
        get() = !saving && name.isNotBlank() && type in validType

    init {
        viewModelScope.launch { categories.ensureLoaded() }
    }

    fun refresh() {
        viewModelScope.launch { categories.refresh() }
    }

    fun save() {
        val trimmed = name.trim()
        if (saving || trimmed.isBlank() || type !in validType) {
            error = "Isi nama & pilih tipe."
            return
        }
        saving = true
        error = null
        viewModelScope.launch {
            try {
                categories.createCategory(name = trimmed, categoryType = type)
                name = "" // reset form; list ikut ter-refresh oleh createCategory().
                saving = false
            } catch (e: Exception) {
                saving = false
                error = "Gagal menyimpan kategori."
            }
        }
    }

    companion object {
        fun factory(categories: CategoriesRepository) = viewModelFactory {
            initializer { CategoryViewModel(categories) }
        }
    }
}
