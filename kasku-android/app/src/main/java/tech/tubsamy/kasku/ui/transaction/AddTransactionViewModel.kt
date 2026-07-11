package tech.tubsamy.kasku.ui.transaction

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.AccountItem
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.CategoriesRepository
import tech.tubsamy.kasku.data.CategoryItem
import tech.tubsamy.kasku.data.TransactionsRepository
import tech.tubsamy.kasku.data.sync.TransactionMutations
import java.time.LocalDate

/**
 * Dipakai untuk Tambah DAN Edit transaksi (reuse layar sama). editId != null → mode edit:
 * prefill dari Room lalu save() memanggil mutations.update; null → create baru.
 */
class AddTransactionViewModel(
    repo: AccountsRepository,
    private val mutations: TransactionMutations,
    private val categories: CategoriesRepository,
    private val transactions: TransactionsRepository,
    private val editId: String? = null,
    private val today: () -> String = { LocalDate.now().toString() },
) : ViewModel() {

    val isEdit: Boolean get() = editId != null

    val accounts: StateFlow<List<AccountItem>> =
        repo.observeAccounts().stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    var amount by mutableStateOf("") // string, diparse ke Long (rupiah bulat)
    var notes by mutableStateOf("")
    var accountId by mutableStateOf<String?>(null)
    var categoryId by mutableStateOf<String?>(null)      // OPSIONAL — boleh null
    var date by mutableStateOf(today())                  // "YYYY-MM-DD", default hari ini
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    // Dropdown kategori reaktif ke tipe terpilih. categories.version memicu re-emit
    // begitu cache in-memory terisi (hindari dropdown kosong nyangkut).
    private val _type = MutableStateFlow("EXPENSE")
    var type: String
        get() = _type.value
        set(value) {
            if (_type.value != value) {
                _type.value = value
                categoryId = null // reset: kategori valid untuk satu tipe saja
            }
        }

    val categoryOptions: StateFlow<List<CategoryItem>> =
        kotlinx.coroutines.flow.combine(_type, categories.version) { t, _ -> categories.forType(t) }
            .stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    private val amountValue: Long?
        get() = amount.filter { it.isDigit() }.toLongOrNull()

    val canSave: Boolean
        get() = !saving && accountId != null && (amountValue ?: 0) > 0

    init {
        viewModelScope.launch { categories.ensureLoaded() }
        if (editId != null) {
            viewModelScope.launch {
                transactions.findRaw(editId)?.let { tx ->
                    // Set _type langsung (bukan via setter) agar categoryId prefill tak ter-reset.
                    _type.value = tx.transaction_type
                    amount = tx.amount_idr.toString()
                    notes = tx.notes ?: ""
                    accountId = tx.account_id
                    categoryId = tx.category_id
                    date = tx.transaction_date
                }
            }
        }
    }

    fun save(onDone: () -> Unit) {
        val acc = accountId
        val amt = amountValue
        if (saving || acc == null || amt == null || amt <= 0) {
            error = "Isi jumlah & pilih akun."
            return
        }
        saving = true
        error = null
        viewModelScope.launch {
            try {
                if (editId != null) {
                    // categoryId bisa null (Tanpa kategori) — update() abaikan null sbg "pertahankan".
                    // ponytail: mode edit tak dukung ubah kategori jadi null. Belum diminta; tambah
                    // param clearCategory ke update() bila perlu.
                    mutations.update(
                        id = editId,
                        accountId = acc,
                        type = type,
                        amountIdr = amt,
                        categoryId = categoryId,
                        date = date,
                        notes = notes,
                    )
                } else {
                    mutations.create(
                        accountId = acc,
                        type = type,
                        amountIdr = amt,
                        categoryId = categoryId,
                        date = date,
                        notes = notes,
                    )
                }
                saving = false
                onDone() // optimistic: transaksi masuk Room + antre sync, langsung kembali
            } catch (e: Exception) {
                saving = false
                error = "Gagal menyimpan transaksi."
            }
        }
    }

    companion object {
        fun factory(
            repo: AccountsRepository,
            mutations: TransactionMutations,
            categories: CategoriesRepository,
            transactions: TransactionsRepository,
            editId: String? = null,
        ) = viewModelFactory {
            initializer { AddTransactionViewModel(repo, mutations, categories, transactions, editId) }
        }
    }
}
