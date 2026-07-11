package tech.tubsamy.kasku.data.local

import androidx.room.Entity
import androidx.room.Index
import androidx.room.PrimaryKey

/**
 * Room entities untuk offline-first sync (port dari golden reference web).
 *
 * Field domain memakai penamaan snake_case sesuai jalur SYNC (to_jsonb DB), BUKAN
 * PascalCase REST /accounts. balance/amount = Long (rupiah bulat).
 *
 * Base SyncableEntity fields (id, updatedAt, localDirty, deleted) diulang per tabel
 * — Room tidak mendukung @Embedded inheritance yang ringkas untuk kasus ini, dan
 * eksplisit lebih jelas. ponytail: duplikasi 4 kolom < abstraksi embedded.
 */

@Entity(tableName = "accounts", indices = [Index("updated_at")])
data class AccountEntity(
    @PrimaryKey val id: String,
    val sync_id: String? = null,
    val updated_at: String,
    val local_dirty: Boolean = false,
    val deleted: Boolean = false,
    // domain (snake_case dari sync payload)
    val name: String,
    val account_type: String,
    val balance: Long,
    val currency: String,
)

@Entity(tableName = "transactions", indices = [Index("updated_at")])
data class TransactionEntity(
    @PrimaryKey val id: String,
    val sync_id: String? = null,
    val updated_at: String,
    val local_dirty: Boolean = false,
    val deleted: Boolean = false,
    // domain
    val account_id: String,
    val category_id: String? = null,
    val budget_id: String? = null,
    val transaction_type: String,
    val amount_idr: Long,
    val transaction_date: String,
    val notes: String? = null,
    val to_account_id: String? = null,
)

@Entity(tableName = "investments", indices = [Index("updated_at")])
data class InvestmentEntity(
    @PrimaryKey val id: String,
    val sync_id: String? = null,
    val updated_at: String,
    val local_dirty: Boolean = false,
    val deleted: Boolean = false,
    // domain
    val name: String,
    val asset_type: String,
    val symbol: String? = null,
    val units: Double,
    val avg_buy_price_idr: Long,
)

@Entity(
    tableName = "sync_queue",
    indices = [Index("created_at"), Index("status")],
)
data class SyncQueueEntity(
    @PrimaryKey val sync_id: String,
    val resource: String,       // accounts | transactions | investments
    val operation: String,      // CREATE | UPDATE | DELETE
    val entity_id: String,
    val payload: String,        // JSON serialized entity
    val status: String,         // PENDING | IN_FLIGHT | FAILED
    val attempts: Int = 0,
    val last_error: String? = null,
    val created_at: String,
)

@Entity(tableName = "sync_conflicts", indices = [Index("detected_at")])
data class SyncConflictEntity(
    @PrimaryKey val id: String,
    val resource: String,
    val entity_id: String,
    val origin: String,         // pull | push
    val local_value: String,    // JSON
    val server_value: String,   // JSON
    val detected_at: String,
)

@Entity(tableName = "sync_meta")
data class SyncMetaEntity(
    @PrimaryKey val resource: String,
    val last_synced_at: String, // ISO 8601 server timestamp dari pull terakhir
)
