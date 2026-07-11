package tech.tubsamy.kasku.data.sync

import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull
import tech.tubsamy.kasku.data.local.AccountEntity
import tech.tubsamy.kasku.data.local.InvestmentEntity
import tech.tubsamy.kasku.data.local.TransactionEntity

/** Resource lokal ↔ entity_type server (port RESOURCE_TO_SERVER_ENTITY). */
enum class Resource(val local: String, val server: String) {
    ACCOUNTS("accounts", "financial_account"),
    TRANSACTIONS("transactions", "transaction"),
    INVESTMENTS("investments", "investment_asset");

    companion object {
        fun fromServer(entityType: String): Resource? = entries.find { it.server == entityType }
    }
}

/** Semua resource yang disinkron (urutan pull). */
val SYNCABLE_RESOURCES: List<Resource> = Resource.entries

/**
 * Mapping JSON sync (snake_case dari to_jsonb) → Room entity.
 * localDirty selalu false: data dari server sudah "bersih".
 */
object SyncMapping {

    private fun JsonObject.str(key: String, default: String = ""): String =
        this[key]?.jsonPrimitive?.content ?: default

    private fun JsonObject.strOrNull(key: String): String? =
        this[key]?.jsonPrimitive?.let { if (it.content == "null") null else it.content }

    private fun JsonObject.longVal(key: String, default: Long = 0): Long =
        this[key]?.jsonPrimitive?.longOrNull ?: default

    private fun JsonObject.doubleVal(key: String, default: Double = 0.0): Double =
        this[key]?.jsonPrimitive?.content?.toDoubleOrNull() ?: default

    private fun JsonObject.deleted(): Boolean =
        this["_deleted"]?.jsonPrimitive?.booleanOrNull
            ?: this["deleted"]?.jsonPrimitive?.booleanOrNull
            ?: false

    fun toAccount(data: JsonElement): AccountEntity {
        val o = data.jsonObject
        return AccountEntity(
            id = o.str("id"),
            sync_id = o.strOrNull("sync_id"),
            updated_at = o.str("updated_at"),
            local_dirty = false,
            deleted = o.deleted(),
            name = o.str("name"),
            account_type = o.str("account_type"),
            balance = o.longVal("balance"),
            currency = o.str("currency", "IDR"),
        )
    }

    fun toTransaction(data: JsonElement): TransactionEntity {
        val o = data.jsonObject
        return TransactionEntity(
            id = o.str("id"),
            sync_id = o.strOrNull("sync_id"),
            updated_at = o.str("updated_at"),
            local_dirty = false,
            deleted = o.deleted(),
            account_id = o.str("account_id"),
            category_id = o.strOrNull("category_id"),
            budget_id = o.strOrNull("budget_id"),
            transaction_type = o.str("transaction_type"),
            amount_idr = o.longVal("amount_idr"),
            transaction_date = o.str("transaction_date"),
            notes = o.strOrNull("notes"),
            to_account_id = o.strOrNull("to_account_id"),
        )
    }

    fun toInvestment(data: JsonElement): InvestmentEntity {
        val o = data.jsonObject
        return InvestmentEntity(
            id = o.str("id"),
            sync_id = o.strOrNull("sync_id"),
            updated_at = o.str("updated_at"),
            local_dirty = false,
            deleted = o.deleted(),
            name = o.str("name"),
            asset_type = o.str("asset_type"),
            symbol = o.strOrNull("symbol"),
            units = o.doubleVal("units"),
            avg_buy_price_idr = o.longVal("avg_buy_price_idr"),
        )
    }

    /** Ambil (updated_at, deleted) dari JSON server untuk LWW tanpa mapping penuh. */
    fun serverView(data: JsonElement): SyncableView {
        val o = data.jsonObject
        return SyncableView(
            updatedAt = o.str("updated_at"),
            deleted = o.deleted(),
            localDirty = false,
        )
    }
}
