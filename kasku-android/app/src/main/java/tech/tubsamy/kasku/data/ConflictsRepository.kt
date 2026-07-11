package tech.tubsamy.kasku.data

import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import tech.tubsamy.kasku.data.local.KasKuDatabase

/** Konflik sync untuk UI review. local/serverValue = raw JSON string (ditampilkan apa adanya). */
data class ConflictItem(
    val id: String,
    val resource: String, // accounts | transactions | investments
    val entityId: String,
    val origin: String,   // pull | push
    val localValue: String,
    val serverValue: String,
    val detectedAt: String,
)

/**
 * Infra bersama F4 (list Review Konflik) + F5 (badge Home). Sumber = Room sync_conflicts.
 *
 * acceptServer = buang konflik (server value sudah ter-apply di Room via LWW apply_server).
 * keepLocal butuh mutations per-resource → didelegasikan ke ConflictResolutionService.
 */
class ConflictsRepository(
    db: KasKuDatabase,
) {
    private val dao = db.syncConflictDao()

    fun observeAll(): Flow<List<ConflictItem>> =
        dao.observe().map { rows ->
            rows.map {
                ConflictItem(
                    id = it.id,
                    resource = it.resource,
                    entityId = it.entity_id,
                    origin = it.origin,
                    localValue = it.local_value,
                    serverValue = it.server_value,
                    detectedAt = it.detected_at,
                )
            }
        }

    /** Badge Home: jumlah konflik reaktif. */
    fun observeCount(): Flow<Int> = dao.observeCount()

    /** "Terima server": buang konflik. Server value sudah ada di Room. */
    suspend fun acceptServer(id: String) = dao.deleteById(id)
}
