package tech.tubsamy.kasku.ui.conflicts

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.ConflictItem
import tech.tubsamy.kasku.data.ConflictsRepository
import tech.tubsamy.kasku.data.sync.ConflictResolutionService

/**
 * F4 Review Konflik — list sync_conflicts dari Room. Aksi:
 *  - keepLocal ("Pakai versi saya"): re-enqueue nilai lokal → menang LWW → hapus konflik.
 *  - acceptServer ("Terima server"): hapus konflik (server value sudah ter-apply di Room).
 */
class ConflictsViewModel(
    private val conflicts: ConflictsRepository,
    private val resolution: ConflictResolutionService,
) : ViewModel() {

    val items: StateFlow<List<ConflictItem>> =
        conflicts.observeAll().stateIn(
            scope = viewModelScope,
            started = SharingStarted.WhileSubscribed(5_000),
            initialValue = emptyList(),
        )

    fun keepLocal(conflict: ConflictItem) = viewModelScope.launch {
        resolution.keepLocal(conflict)
    }

    fun acceptServer(conflict: ConflictItem) = viewModelScope.launch {
        conflicts.acceptServer(conflict.id)
    }

    companion object {
        fun factory(
            conflicts: ConflictsRepository,
            resolution: ConflictResolutionService,
        ) = viewModelFactory {
            initializer { ConflictsViewModel(conflicts, resolution) }
        }
    }
}
