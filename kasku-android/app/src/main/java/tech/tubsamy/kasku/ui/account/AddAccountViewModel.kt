package tech.tubsamy.kasku.ui.account

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.AccountsRepository

/**
 * Form tambah/edit akun (ONLINE). editId != null → prefill dari cache AccountsRepository
 * (refresh dulu supaya cache terisi bila layar dibuka langsung). Mutasi lewat REST; sukses →
 * repo menyegarkan cache. Currency tetap IDR — ponytail: satu mata uang untuk sekarang.
 */
class AddAccountViewModel(
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
                accounts.refresh()
                val acc = accounts.find(id)
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
                    // Saldo tidak diubah saat edit — backend mengabaikannya (saldo bergerak lewat transaksi).
                    accounts.update(editId, name = name.trim(), accountType = accountType)
                } else {
                    accounts.create(name = name.trim(), accountType = accountType, balance = balanceValue)
                }
                saving = false
                onSaved()
            } catch (e: Exception) {
                saving = false
                error = "Gagal menyimpan akun. Periksa koneksi."
            }
        }
    }

    companion object {
        fun factory(
            accounts: AccountsRepository,
            editId: String? = null,
        ) = viewModelFactory {
            initializer { AddAccountViewModel(accounts, editId) }
        }
    }
}
