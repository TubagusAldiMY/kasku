package tech.tubsamy.kasku.ui.investment

import android.content.Context
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.stateIn
import tech.tubsamy.kasku.data.InvestmentItem
import tech.tubsamy.kasku.data.InvestmentsRepository
import tech.tubsamy.kasku.data.sync.SyncWorker

/**
 * OFFLINE-FIRST: investasi dibaca reaktif dari Room (diisi sync). Masuk layar → picu
 * sync sekali (one-off). Pola sama dengan HomeViewModel.
 */
class InvestmentViewModel(
    repo: InvestmentsRepository,
    private val appContext: Context,
) : ViewModel() {

    val investments: StateFlow<List<InvestmentItem>> =
        repo.observeInvestments().stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = emptyList(),
        )

    init {
        SyncWorker.enqueueOnce(appContext)
    }

    fun refresh() = SyncWorker.enqueueOnce(appContext)

    companion object {
        fun factory(
            repo: InvestmentsRepository,
            appContext: Context,
        ) = viewModelFactory {
            initializer { InvestmentViewModel(repo, appContext.applicationContext) }
        }
    }
}
