package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import tech.tubsamy.kasku.data.remote.FinanceApi
import tech.tubsamy.kasku.data.remote.dto.CreateCategoryRequest

/** Kategori read-only untuk UI (nama + tipe filter dropdown). */
data class CategoryItem(
    val id: String,
    val name: String,
    val categoryType: String, // INCOME | EXPENSE | BOTH
)

/**
 * Infra bersama F2 (nama kategori riwayat) + F3 (dropdown AddTransaction).
 *
 * Kategori TIDAK disimpan di Room — di-fetch online (GET /v1/categories) lalu cache
 * in-memory. KMP-ready: hanya Retrofit interface + Mutex, tanpa import Android.
 *
 * Offline aman: fetch gagal → cache tetap null → nameFor()/forType() fallback (tak throw).
 */
class CategoriesRepository(
    private val financeApi: FinanceApi,
) {
    // null = belum pernah fetch sukses. @Volatile: dibaca non-suspend saat map Flow.
    @Volatile
    private var cache: List<CategoryItem>? = null
    private val fetchLock = Mutex()

    private val _version = MutableStateFlow(0)

    /**
     * Naik tiap fetch sukses. TransactionsRepository combine dengan ini supaya list
     * riwayat re-emit dengan nama benar setelah cache in-memory terisi (hindari "Umum" nyangkut).
     */
    val version: StateFlow<Int> = _version

    /** Fetch sekali; sekali sukses tak fetch lagi (kecuali refresh()). Gagal → diam, cache null. */
    suspend fun ensureLoaded() {
        if (cache != null) return
        fetchLock.withLock {
            if (cache != null) return
            fetchInto()
        }
    }

    /** Force re-fetch (mis. pull-to-refresh). */
    suspend fun refresh() {
        fetchLock.withLock { fetchInto() }
    }

    /**
     * Buat kategori baru (POST /v1/categories) lalu refresh cache supaya dropdown/daftar
     * ikut terisi. name di-trim; categoryType harus INCOME|EXPENSE|BOTH (divalidasi caller/backend).
     * Throw bila gagal (offline/validasi backend) — VM yang menampilkan errornya.
     */
    suspend fun createCategory(name: String, categoryType: String) {
        financeApi.createCategory(CreateCategoryRequest(name = name.trim(), categoryType = categoryType))
        refresh() // re-fetch: cache + version naik → observer layar kelola & dropdown ikut update.
    }

    /** Semua kategori aktif untuk layar kelola. Kosong bila cache belum terisi. */
    fun all(): List<CategoryItem> = cache ?: emptyList()

    private suspend fun fetchInto() {
        val result = runCatching {
            val envelope = financeApi.listCategories()
            // ponytail: buang isDeleted di fetch → nameFor & forType tak perlu simpan flag.
            (envelope.data ?: emptyList())
                .filterNot { it.isDeleted }
                .map { CategoryItem(id = it.id, name = it.name, categoryType = it.categoryType) }
        }
        // ponytail: gagal (offline / 5xx) → biarkan cache apa adanya, UI fallback "Umum".
        val loaded = result.getOrNull() ?: return
        cache = loaded
        _version.value += 1
    }

    /** Nama untuk 1 id; "Umum" bila offline/null/tak ketemu. Sinkron, tak pernah throw. */
    fun nameFor(categoryId: String?): String {
        if (categoryId.isNullOrBlank()) return "Umum"
        return cache?.firstOrNull { it.id == categoryId }?.name ?: "Umum"
    }

    /**
     * Item dropdown AddTransaction untuk tipe terpilih (INCOME/EXPENSE):
     * categoryType == type || == "BOTH". Kosong bila cache belum terisi.
     */
    fun forType(type: String): List<CategoryItem> {
        val items = cache ?: return emptyList()
        return items.filter { it.categoryType == type || it.categoryType == "BOTH" }
    }
}
