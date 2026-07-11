package tech.tubsamy.kasku.data.sync

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import tech.tubsamy.kasku.data.local.AccountEntity
import tech.tubsamy.kasku.data.local.KasKuDatabase
import tech.tubsamy.kasku.data.local.SyncQueueEntity
import java.util.UUID

/**
 * Optimistic mutations untuk accounts (port mutations.ts, accounts-only scope M2).
 * Pola: tulis Room dulu (localDirty=true) → enqueue sync_queue → trigger sync di
 * background (fire-and-forget). Setiap mutasi generate sync_id BARU (idempotensi).
 *
 * ponytail: accounts-only. Transaksi/investasi menyusul — entity+DAO sudah ada.
 */
class AccountMutations(
    private val db: KasKuDatabase,
    private val json: Json,
    private val clock: () -> String = { java.time.Instant.now().toString() },
    private val fireSync: () -> Unit,
) {
    private val accountDao = db.accountDao()
    private val queueDao = db.syncQueueDao()

    suspend fun create(name: String, accountType: String, balance: Long, currency: String = "IDR"): String {
        val id = UUID.randomUUID().toString()
        val syncId = UUID.randomUUID().toString()
        val now = clock()
        val row = AccountEntity(
            id = id,
            sync_id = syncId,
            updated_at = now,
            local_dirty = true,
            deleted = false,
            name = name,
            account_type = accountType,
            balance = balance,
            currency = currency,
        )
        accountDao.upsert(row)
        enqueue(syncId, "CREATE", id, accountPayload(row))
        fireSync()
        return id
    }

    suspend fun update(
        id: String,
        name: String? = null,
        accountType: String? = null,
        balance: Long? = null,
        currency: String? = null,
    ) {
        val existing = accountDao.findById(id) ?: return
        val syncId = UUID.randomUUID().toString()
        val now = clock()
        val row = existing.copy(
            sync_id = syncId,
            updated_at = now,
            local_dirty = true,
            name = name ?: existing.name,
            account_type = accountType ?: existing.account_type,
            balance = balance ?: existing.balance,
            currency = currency ?: existing.currency,
        )
        accountDao.upsert(row)
        enqueue(syncId, "UPDATE", id, accountPayload(row))
        fireSync()
    }

    suspend fun delete(id: String) {
        val existing = accountDao.findById(id) ?: return
        val syncId = UUID.randomUUID().toString()
        val now = clock()
        // Soft delete: tombstone tetap ada untuk sinkronisasi.
        accountDao.upsert(existing.copy(sync_id = syncId, updated_at = now, local_dirty = true, deleted = true))
        val payload = buildJsonObject { put("id", JsonPrimitive(id)) }
        enqueue(syncId, "DELETE", id, payload)
        fireSync()
    }

    private suspend fun enqueue(syncId: String, operation: String, entityId: String, payload: JsonObject) {
        queueDao.enqueue(
            SyncQueueEntity(
                sync_id = syncId,
                resource = Resource.ACCOUNTS.local,
                operation = operation,
                entity_id = entityId,
                payload = json.encodeToString(JsonObject.serializer(), payload),
                status = "PENDING",
                attempts = 0,
                last_error = null,
                created_at = clock(),
            ),
        )
    }

    /** Payload push = snake_case sesuai jalur sync (bukan PascalCase REST). */
    private fun accountPayload(row: AccountEntity): JsonObject = buildJsonObject {
        put("id", JsonPrimitive(row.id))
        put("name", JsonPrimitive(row.name))
        put("account_type", JsonPrimitive(row.account_type))
        put("balance", JsonPrimitive(row.balance))
        put("currency", JsonPrimitive(row.currency))
        put("updated_at", JsonPrimitive(row.updated_at))
    }
}
