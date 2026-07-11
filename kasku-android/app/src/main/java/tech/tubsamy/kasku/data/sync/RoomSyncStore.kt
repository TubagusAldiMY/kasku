package tech.tubsamy.kasku.data.sync

import kotlinx.serialization.json.JsonElement
import tech.tubsamy.kasku.data.local.KasKuDatabase
import tech.tubsamy.kasku.data.local.SyncConflictEntity
import tech.tubsamy.kasku.data.local.SyncMetaEntity
import java.util.UUID

/** Implementasi [SyncStore] di atas Room. Satu-satunya jembatan engine ↔ DB. */
class RoomSyncStore(
    private val db: KasKuDatabase,
    private val clock: () -> String = { java.time.Instant.now().toString() },
) : SyncStore {

    private val accountDao = db.accountDao()
    private val transactionDao = db.transactionDao()
    private val investmentDao = db.investmentDao()
    private val queueDao = db.syncQueueDao()
    private val conflictDao = db.syncConflictDao()
    private val metaDao = db.syncMetaDao()

    override suspend fun lastSyncedAt(resource: Resource): String? =
        metaDao.lastSyncedAt(resource.local)

    override suspend fun setLastSyncedAt(resource: Resource, iso: String) =
        metaDao.upsert(SyncMetaEntity(resource.local, iso))

    override suspend fun localView(resource: Resource, id: String): SyncableView? =
        when (resource) {
            Resource.ACCOUNTS -> accountDao.findById(id)?.let {
                SyncableView(it.updated_at, it.deleted, it.local_dirty)
            }
            Resource.TRANSACTIONS -> transactionDao.findById(id)?.let {
                SyncableView(it.updated_at, it.deleted, it.local_dirty)
            }
            Resource.INVESTMENTS -> investmentDao.findById(id)?.let {
                SyncableView(it.updated_at, it.deleted, it.local_dirty)
            }
        }

    override suspend fun upsertFromServer(resource: Resource, data: JsonElement) {
        when (resource) {
            Resource.ACCOUNTS -> accountDao.upsert(SyncMapping.toAccount(data))
            Resource.TRANSACTIONS -> transactionDao.upsert(SyncMapping.toTransaction(data))
            Resource.INVESTMENTS -> investmentDao.upsert(SyncMapping.toInvestment(data))
        }
    }

    override suspend fun hardDelete(resource: Resource, id: String) {
        when (resource) {
            Resource.ACCOUNTS -> accountDao.hardDelete(id)
            Resource.TRANSACTIONS -> transactionDao.hardDelete(id)
            Resource.INVESTMENTS -> investmentDao.hardDelete(id)
        }
    }

    override suspend fun pendingOps(limit: Int): List<QueuedOp> =
        queueDao.pending(limit).mapNotNull { row ->
            val resource = SYNCABLE_RESOURCES.find { it.local == row.resource } ?: return@mapNotNull null
            QueuedOp(row.sync_id, resource, row.operation, row.entity_id, row.payload)
        }

    override suspend fun markInFlight(syncIds: List<String>) = queueDao.markInFlight(syncIds)
    override suspend fun markAcked(syncId: String) = queueDao.markAcked(syncId)
    override suspend fun markFailed(syncId: String, error: String) = queueDao.markFailed(syncId, error)
    override suspend fun recoverStuck() = queueDao.recoverStuck()
    override suspend fun queuedCount(): Int = queueDao.count()

    override suspend fun recordConflict(
        resource: Resource,
        entityId: String,
        origin: String,
        localValue: String,
        serverValue: String,
    ) = conflictDao.insert(
        SyncConflictEntity(
            id = UUID.randomUUID().toString(),
            resource = resource.local,
            entity_id = entityId,
            origin = origin,
            local_value = localValue,
            server_value = serverValue,
            detected_at = clock(),
        ),
    )

    override suspend fun conflictCount(): Int = conflictDao.count()
}
