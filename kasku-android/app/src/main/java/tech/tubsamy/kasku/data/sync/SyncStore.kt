package tech.tubsamy.kasku.data.sync

import kotlinx.serialization.json.JsonElement

/**
 * Port yang dipakai SyncEngine untuk berbicara ke persistensi lokal.
 * Interface (bukan Room langsung) supaya engine bebas import Android → testable
 * dengan fake in-memory (lihat FakeSyncStore di test).
 */
interface SyncStore {

    // sync_meta
    suspend fun lastSyncedAt(resource: Resource): String?
    suspend fun setLastSyncedAt(resource: Resource, iso: String)

    // entity local (LWW butuh view lokal; upsert/hardDelete menerapkan keputusan)
    suspend fun localView(resource: Resource, id: String): SyncableView?
    suspend fun upsertFromServer(resource: Resource, data: JsonElement)
    suspend fun hardDelete(resource: Resource, id: String)

    // sync_queue
    suspend fun pendingOps(limit: Int): List<QueuedOp>
    suspend fun markInFlight(syncIds: List<String>)
    suspend fun markAcked(syncId: String)
    suspend fun markFailed(syncId: String, error: String)
    suspend fun recoverStuck()
    suspend fun queuedCount(): Int

    // sync_conflicts
    suspend fun recordConflict(
        resource: Resource,
        entityId: String,
        origin: String,          // pull | push
        localValue: String,      // JSON
        serverValue: String,     // JSON
    )
    suspend fun conflictCount(): Int
}

/** Baris queue yang sudah siap dikirim (payload sudah JSON). */
data class QueuedOp(
    val syncId: String,
    val resource: Resource,
    val operation: String,       // CREATE | UPDATE | DELETE
    val entityId: String,
    val payloadJson: String,
)
