package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import tech.tubsamy.kasku.data.remote.FinanceApi
import tech.tubsamy.kasku.data.remote.dto.CreateAccountRequest
import tech.tubsamy.kasku.data.remote.dto.UpdateAccountRequest

/** Item akun untuk UI. */
data class AccountItem(
    val id: String,
    val name: String,
    val accountType: String,
    val balance: Long,
    val currency: String,
)

/**
 * ONLINE (pola BudgetsRepository): fetch GET /accounts → cache in-memory StateFlow.
 * Mutasi CRUD memanggil REST lalu refresh(); gagal fetch → cache lama dipertahankan,
 * gagal mutasi → throw (VM tampilkan error).
 */
class AccountsRepository(
    private val financeApi: FinanceApi,
) {
    private val _accounts = MutableStateFlow<List<AccountItem>>(emptyList())
    val accounts: StateFlow<List<AccountItem>> = _accounts

    /** Ambil dari cache tanpa fetch (prefill layar edit). */
    fun find(id: String): AccountItem? = _accounts.value.firstOrNull { it.id == id }

    /** Fetch ulang; offline/5xx → diam, nilai lama tetap tampil. */
    suspend fun refresh() {
        val loaded = runCatching {
            (financeApi.listAccounts().data ?: emptyList()).map {
                AccountItem(
                    id = it.id,
                    name = it.name,
                    accountType = it.accountType,
                    balance = it.balance,
                    currency = it.currency,
                )
            }
        }.getOrNull() ?: return
        _accounts.value = loaded
    }

    suspend fun create(name: String, accountType: String, balance: Long) {
        financeApi.createAccount(
            CreateAccountRequest(
                name = name.trim(),
                accountType = accountType,
                balance = balance,
                initialBalance = balance,
            ),
        )
        refresh()
    }

    /** Saldo tidak ikut diubah — backend hanya menerima nama & tipe (saldo bergerak lewat transaksi). */
    suspend fun update(id: String, name: String, accountType: String) {
        financeApi.updateAccount(id, UpdateAccountRequest(name = name.trim(), accountType = accountType))
        refresh()
    }

    suspend fun delete(id: String) {
        financeApi.deleteAccount(id)
        refresh()
    }
}
