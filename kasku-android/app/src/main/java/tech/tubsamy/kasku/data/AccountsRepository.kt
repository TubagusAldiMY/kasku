package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import tech.tubsamy.kasku.data.local.KasKuDatabase

/** Item akun untuk UI (dari Room, hasil sinkronisasi). */
data class AccountItem(
    val id: String,
    val name: String,
    val accountType: String,
    val balance: Long,
    val currency: String,
)

/**
 * OFFLINE-FIRST (M2): sumber kebenaran tampilan = Room, diisi oleh SyncEngine.
 * Repo ini TIDAK lagi memanggil REST /accounts — sync yang mengisi tabel.
 */
class AccountsRepository(
    db: KasKuDatabase,
) {
    private val accountDao = db.accountDao()

    /** Akun aktif, reaktif. Kosong sampai sync pertama mengisi Room. */
    fun observeAccounts(): Flow<List<AccountItem>> =
        accountDao.observeActive().map { rows ->
            rows.map { AccountItem(it.id, it.name, it.account_type, it.balance, it.currency) }
        }
}
