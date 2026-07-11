package tech.tubsamy.kasku.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import kotlinx.coroutines.flow.Flow

@Dao
interface AccountDao {
    /** Akun aktif (belum di-soft-delete) untuk tampilan Home, reaktif. */
    @Query("SELECT * FROM accounts WHERE deleted = 0 ORDER BY name")
    fun observeActive(): Flow<List<AccountEntity>>

    @Query("SELECT * FROM accounts WHERE id = :id")
    suspend fun findById(id: String): AccountEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(row: AccountEntity)

    @Query("DELETE FROM accounts WHERE id = :id")
    suspend fun hardDelete(id: String)
}

@Dao
interface TransactionDao {
    @Query("SELECT * FROM transactions WHERE id = :id")
    suspend fun findById(id: String): TransactionEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(row: TransactionEntity)

    @Query("DELETE FROM transactions WHERE id = :id")
    suspend fun hardDelete(id: String)
}

@Dao
interface InvestmentDao {
    @Query("SELECT * FROM investments WHERE id = :id")
    suspend fun findById(id: String): InvestmentEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(row: InvestmentEntity)

    @Query("DELETE FROM investments WHERE id = :id")
    suspend fun hardDelete(id: String)
}

@Dao
interface SyncQueueDao {
    /** Batch untuk push: skip IN_FLIGHT (recovery dari crash tetap terpisah). */
    @Query("SELECT * FROM sync_queue WHERE status != 'IN_FLIGHT' ORDER BY created_at LIMIT :limit")
    suspend fun pending(limit: Int): List<SyncQueueEntity>

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun enqueue(row: SyncQueueEntity)

    @Query("UPDATE sync_queue SET status = 'IN_FLIGHT' WHERE sync_id IN (:syncIds)")
    suspend fun markInFlight(syncIds: List<String>)

    @Query("DELETE FROM sync_queue WHERE sync_id = :syncId")
    suspend fun markAcked(syncId: String)

    @Query(
        "UPDATE sync_queue SET status = 'FAILED', attempts = attempts + 1, last_error = :error " +
            "WHERE sync_id = :syncId",
    )
    suspend fun markFailed(syncId: String, error: String)

    /** Recovery yatim: IN_FLIGHT (crash saat push) → PENDING. */
    @Query("UPDATE sync_queue SET status = 'PENDING' WHERE status = 'IN_FLIGHT'")
    suspend fun recoverStuck()

    @Query("SELECT COUNT(*) FROM sync_queue")
    suspend fun count(): Int
}

@Dao
interface SyncConflictDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(row: SyncConflictEntity)

    @Query("SELECT COUNT(*) FROM sync_conflicts")
    suspend fun count(): Int
}

@Dao
interface SyncMetaDao {
    @Query("SELECT last_synced_at FROM sync_meta WHERE resource = :resource")
    suspend fun lastSyncedAt(resource: String): String?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(row: SyncMetaEntity)
}
