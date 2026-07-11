package tech.tubsamy.kasku.ui.transaction

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.AccountItem
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.sync.TransactionMutations

class AddTransactionViewModel(
    repo: AccountsRepository,
    private val mutations: TransactionMutations,
) : ViewModel() {

    val accounts: StateFlow<List<AccountItem>> =
        repo.observeAccounts().stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), emptyList())

    var amount by mutableStateOf("") // string, diparse ke Long (rupiah bulat)
    var type by mutableStateOf("EXPENSE") // INCOME | EXPENSE
    var accountId by mutableStateOf<String?>(null)
    var notes by mutableStateOf("")
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    private val amountValue: Long?
        get() = amount.filter { it.isDigit() }.toLongOrNull()

    val canSave: Boolean
        get() = !saving && accountId != null && (amountValue ?: 0) > 0

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
                mutations.create(accountId = acc, type = type, amountIdr = amt, notes = notes)
                saving = false
                onDone() // optimistic: transaksi masuk Room + antre sync, langsung kembali
            } catch (e: Exception) {
                saving = false
                error = "Gagal menyimpan transaksi."
            }
        }
    }

    companion object {
        fun factory(repo: AccountsRepository, mutations: TransactionMutations) = viewModelFactory {
            initializer { AddTransactionViewModel(repo, mutations) }
        }
    }
}
