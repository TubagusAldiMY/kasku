package tech.tubsamy.kasku.data.sync

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import tech.tubsamy.kasku.data.local.InvestmentEntity
import tech.tubsamy.kasku.data.local.KasKuDatabase
import tech.tubsamy.kasku.data.local.SyncQueueEntity
import java.util.UUID

/**
 * Optimistic mutations untuk investasi (pola sama dgn AccountMutations/TransactionMutations).
 * Tulis Room (local_dirty=true) → enqueue sync_queue → trigger sync. Payload snake_case
 * = golden ref web (InvestmentRow): name/asset_type/symbol?/units/avg_buy_price_idr.
 */
class InvestmentMutations(
    private val db: KasKuDatabase,
    private val json: Json,
    private val fireSync: () -> Unit,
    private val clock: () -> String = { java.time.Instant.now().toString() },
) {
    private val investmentDao = db.investmentDao()
    private val queueDao = db.syncQueueDao()

    /** assetType: CRYPTO | GOLD | STOCK | MUTUAL_FUND. symbol opsional (blank → null). */
    suspend fun create(
        name: String,
        assetType: String,
        units: Double,
        avgBuyPriceIdr: Long,
        symbol: String? = null,
    ): String {
        val id = UUID.randomUUID().toString()
        val syncId = UUID.randomUUID().toString()
        val now = clock()
        val row = InvestmentEntity(
            id = id,
            sync_id = syncId,
            updated_at = now,
            local_dirty = true,
            deleted = false,
            name = name,
            asset_type = assetType,
            symbol = symbol?.takeIf { it.isNotBlank() },
            units = units,
            avg_buy_price_idr = avgBuyPriceIdr,
        )
        investmentDao.upsert(row)
        enqueue(syncId, "CREATE", id, payload(row))
        fireSync()
        return id
    }

    private suspend fun enqueue(syncId: String, operation: String, entityId: String, payload: JsonObject) {
        queueDao.enqueue(
            SyncQueueEntity(
                sync_id = syncId,
                resource = Resource.INVESTMENTS.local,
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

    private fun payload(row: InvestmentEntity): JsonObject = buildJsonObject {
        put("id", JsonPrimitive(row.id))
        put("name", JsonPrimitive(row.name))
        put("asset_type", JsonPrimitive(row.asset_type))
        put("units", JsonPrimitive(row.units))
        put("avg_buy_price_idr", JsonPrimitive(row.avg_buy_price_idr))
        put("updated_at", JsonPrimitive(row.updated_at))
        row.symbol?.let { put("symbol", JsonPrimitive(it)) }
    }
}
