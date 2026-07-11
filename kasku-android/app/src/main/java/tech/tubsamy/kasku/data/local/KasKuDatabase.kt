package tech.tubsamy.kasku.data.local

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase

@Database(
    entities = [
        AccountEntity::class,
        TransactionEntity::class,
        InvestmentEntity::class,
        SyncQueueEntity::class,
        SyncConflictEntity::class,
        SyncMetaEntity::class,
    ],
    version = 1,
    exportSchema = false,
)
abstract class KasKuDatabase : RoomDatabase() {
    abstract fun accountDao(): AccountDao
    abstract fun transactionDao(): TransactionDao
    abstract fun investmentDao(): InvestmentDao
    abstract fun syncQueueDao(): SyncQueueDao
    abstract fun syncConflictDao(): SyncConflictDao
    abstract fun syncMetaDao(): SyncMetaDao

    companion object {
        fun build(context: Context): KasKuDatabase =
            Room.databaseBuilder(
                context.applicationContext,
                KasKuDatabase::class.java,
                "kasku.db",
            ).build()
    }
}
