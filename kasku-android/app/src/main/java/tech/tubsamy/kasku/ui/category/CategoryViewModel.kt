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

    // ─── Edit / Hapus (dialog) ──────────────────────────────────────────────────
    // editing != null → dialog terbuka. editBusy dipakai save & hapus (satu tombol aktif).

    var editing by mutableStateOf<CategoryItem?>(null)
        private set
    var editName by mutableStateOf("")
    var editType by mutableStateOf("EXPENSE")
    var editBusy by mutableStateOf(false)
        private set
    var editError by mutableStateOf<String?>(null)
        private set

    val canSaveEdit: Boolean
        get() = !editBusy && editName.isNotBlank() && editType in validType

    fun startEdit(cat: CategoryItem) {
        editing = cat
        editName = cat.name
        editType = cat.categoryType
        editError = null
    }

    fun cancelEdit() {
        if (editBusy) return
        editing = null
        editError = null
    }

    fun saveEdit() {
        val cat = editing ?: return
        val trimmed = editName.trim()
        if (editBusy || trimmed.isBlank() || editType !in validType) {
            editError = "Isi nama & pilih tipe."
            return
        }
        editBusy = true
        editError = null
        viewModelScope.launch {
            try {
                categories.updateCategory(id = cat.id, name = trimmed, categoryType = editType)
                editBusy = false
                editing = null // list ikut ter-refresh oleh updateCategory().
            } catch (e: Exception) {
                editBusy = false
                editError = "Gagal memperbarui kategori."
            }
        }
    }

    fun deleteEdit() {
        val cat = editing ?: return
        if (editBusy) return
        editBusy = true
        editError = null
        viewModelScope.launch {
            try {
                categories.deleteCategory(cat.id)
                editBusy = false
                editing = null
            } catch (e: Exception) {
                editBusy = false
                // Backend tolak 409 bila kategori default / masih dipakai transaksi.
                editError = "Gagal menghapus. Kategori mungkin bawaan atau masih dipakai transaksi."
            }
        }
    }

    companion object {
        fun factory(categories: CategoriesRepository) = viewModelFactory {
            initializer { CategoryViewModel(categories) }
        }
    }
}
