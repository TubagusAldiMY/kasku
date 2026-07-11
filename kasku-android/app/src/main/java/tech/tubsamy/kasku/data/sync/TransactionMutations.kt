package tech.tubsamy.kasku.data.sync

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import tech.tubsamy.kasku.data.local.KasKuDatabase
import tech.tubsamy.kasku.data.local.SyncQueueEntity
import tech.tubsamy.kasku.data.local.TransactionEntity
import java.util.UUID

/**
 * Optimistic mutations untuk transaksi (pola sama dgn AccountMutations).
 * Tulis Room (local_dirty=true) → enqueue sync_queue → trigger sync. Payload snake_case
 * = golden ref web: account_id/transaction_type/amount_idr/transaction_date (+ opsional).
 */
class TransactionMutations(
    private val db: KasKuDatabase,
    private val json: Json,
    private val fireSync: () -> Unit,
    private val clock: () -> String = { java.time.Instant.now().toString() },
    private val today: () -> String = { java.time.LocalDate.now().toString() },
) {
    private val txDao = db.transactionDao()
    private val queueDao = db.syncQueueDao()

    /** date null = hari ini (YYYY-MM-DD). type: INCOME | EXPENSE. categoryId opsional. */
    suspend fun create(
        accountId: String,
        type: String,
        amountIdr: Long,
        categoryId: String? = null,
        date: String? = null,
        notes: String? = null,
    ): String {
        val id = UUID.randomUUID().toString()
        val syncId = UUID.randomUUID().toString()
        val now = clock()
        val row = TransactionEntity(
            id = id,
            sync_id = syncId,
            updated_at = now,
            local_dirty = true,
            deleted = false,
            account_id = accountId,
            category_id = categoryId,
            budget_id = null,
            transaction_type = type,
            amount_idr = amountIdr,
            transaction_date = date ?: today(),
            notes = notes?.takeIf { it.isNotBlank() },
            to_account_id = null,
        )
        txDao.upsert(row)
        enqueue(syncId, "CREATE", id, payload(row))
        fireSync()
        return id
    }

    /**
     * Untuk aksi "Pakai versi saya" (F4): regenerate sync_id + updated_at=now → menang LWW,
     * lalu enqueue UPDATE. Field null = pertahankan nilai existing (pola AccountMutations.update).
     * Edge case: entity lokal sudah hilang → no-op (tak resurrect entity terhapus server).
     */
    suspend fun update(
        id: String,
        accountId: String? = null,
        type: String? = null,
        amountIdr: Long? = null,
        categoryId: String? = null,
        date: String? = null,
        notes: String? = null,
    ) {
        val existing = txDao.findById(id) ?: return
        val syncId = UUID.randomUUID().toString()
        val now = clock()
        val row = existing.copy(
            sync_id = syncId,
            updated_at = now,
            local_dirty = true,
            account_id = accountId ?: existing.account_id,
            category_id = categoryId ?: existing.category_id,
            transaction_type = type ?: existing.transaction_type,
            amount_idr = amountIdr ?: existing.amount_idr,
            transaction_date = date ?: existing.transaction_date,
            notes = notes ?: existing.notes,
        )
        txDao.upsert(row)
        enqueue(syncId, "UPDATE", id, payload(row))
        fireSync()
    }

    /**
     * Soft-delete: tombstone tetap ada untuk sinkronisasi (pola AccountMutations.delete).
     * Regenerate sync_id + updated_at=now → menang LWW; enqueue DELETE payload minimal {id}.
     * Edge case: entity lokal sudah hilang → no-op.
     */
    suspend fun delete(id: String) {
        val existing = txDao.findById(id) ?: return
        val syncId = UUID.randomUUID().toString()
        val now = clock()
        txDao.upsert(existing.copy(sync_id = syncId, updated_at = now, local_dirty = true, deleted = true))
        val payload = buildJsonObject { put("id", JsonPrimitive(id)) }
        enqueue(syncId, "DELETE", id, payload)
        fireSync()
    }

    private suspend fun enqueue(syncId: String, operation: String, entityId: String, payload: JsonObject) {
        queueDao.enqueue(
            SyncQueueEntity(
                sync_id = syncId,
                resource = Resource.TRANSACTIONS.local,
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

    private fun payload(row: TransactionEntity): JsonObject = buildJsonObject {
        put("id", JsonPrimitive(row.id))
        put("account_id", JsonPrimitive(row.account_id))
        put("transaction_type", JsonPrimitive(row.transaction_type))
        put("amount_idr", JsonPrimitive(row.amount_idr))
        put("transaction_date", JsonPrimitive(row.transaction_date))
        put("updated_at", JsonPrimitive(row.updated_at))
        row.category_id?.let { put("category_id", JsonPrimitive(it)) }
        row.notes?.let { put("notes", JsonPrimitive(it)) }
    }
}
