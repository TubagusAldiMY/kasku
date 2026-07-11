package tech.tubsamy.kasku.ui.home

import android.content.Context
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.stateIn
import tech.tubsamy.kasku.data.AccountItem
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.ConflictsRepository
import tech.tubsamy.kasku.data.sync.SyncWorker

/**
 * OFFLINE-FIRST: akun dibaca reaktif dari Room (diisi sync). Masuk Home → picu
 * sync sekali (one-off, NETWORK_CONNECTED). UI selalu punya data lokal terakhir.
 *
 * conflictCount: badge reaktif jumlah konflik sync (F5).
 */
class HomeViewModel(
    repo: AccountsRepository,
    conflicts: ConflictsRepository,
    private val appContext: Context,
) : ViewModel() {

    val accounts: StateFlow<List<AccountItem>> =
        repo.observeAccounts().stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = emptyList(),
        )

    val conflictCount: StateFlow<Int> =
        conflicts.observeCount().stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = 0,
        )

    init {
        // Trigger sync sekali saat masuk Home (mengisi/menyegarkan Room).
        SyncWorker.enqueueOnce(appContext)
    }

    /** Retry manual = picu sync sekali lagi. */
    fun refresh() = SyncWorker.enqueueOnce(appContext)

    companion object {
        fun factory(
            repo: AccountsRepository,
            conflicts: ConflictsRepository,
            appContext: Context,
        ) = viewModelFactory {
            initializer { HomeViewModel(repo, conflicts, appContext.applicationContext) }
        }
    }
}
