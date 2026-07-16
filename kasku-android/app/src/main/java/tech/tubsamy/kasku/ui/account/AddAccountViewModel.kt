package tech.tubsamy.kasku.ui.account

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.sync.AccountMutations

/**
 * Form tambah/edit akun. editId != null → prefill dari snapshot Room (observeAccounts).
 * Mutasi lewat AccountMutations (offline-first: tulis Room + antre sync). Currency
 * tetap IDR — ponytail: satu mata uang untuk sekarang.
 */
class AddAccountViewModel(
    private val mutations: AccountMutations,
    private val accounts: AccountsRepository,
    private val editId: String? = null,
) : ViewModel() {

    val isEdit: Boolean get() = editId != null

    var name by mutableStateOf("")
    var accountType by mutableStateOf("BANK") // BANK | EWALLET | CASH
    var balance by mutableStateOf("") // digit string, rupiah (kosong = 0)
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    private val balanceValue: Long
        get() = balance.filter { it.isDigit() }.toLongOrNull() ?: 0L

    val canSave: Boolean
        get() = !saving && name.isNotBlank()

    init {
        editId?.let { id ->
            viewModelScope.launch {
                val acc = accounts.observeAccounts().first().firstOrNull { it.id == id }
                if (acc == null) {
                    error = "Akun tidak ditemukan."
                } else {
                    name = acc.name
                    accountType = acc.accountType
                    balance = acc.balance.toString()
                }
            }
        }
    }

    fun save(onSaved: () -> Unit) {
        if (!canSave) return
        saving = true
        error = null
        viewModelScope.launch {
            try {
                if (editId != null) {
                    mutations.update(editId, name = name.trim(), accountType = accountType, balance = balanceValue)
                } else {
                    mutations.create(name = name.trim(), accountType = accountType, balance = balanceValue)
                }
                saving = false
                onSaved() // optimistic: masuk Room + antre sync, langsung kembali
            } catch (e: Exception) {
                saving = false
                error = "Gagal menyimpan akun. Periksa koneksi."
            }
        }
    }

    companion object {
        fun factory(
            mutations: AccountMutations,
            accounts: AccountsRepository,
            editId: String? = null,
        ) = viewModelFactory {
            initializer { AddAccountViewModel(mutations, accounts, editId) }
        }
    }
}
