package tech.tubsamy.kasku.data.sync

import android.content.Context
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import tech.tubsamy.kasku.KasKuApplication
import java.util.concurrent.TimeUnit

/**
 * Menjalankan SyncEngine.syncAll() dengan Constraint NETWORK_CONNECTED.
 * Engine diambil dari AppContainer (DI manual) via Application — worker tak boleh
 * merakit graph sendiri.
 *
 * Retry: Result.retry() → WorkManager backoff eksponensial bawaan (default 30s).
 * ponytail: pakai backoff bawaan, cukup untuk sync. Tune jika throughput jadi isu.
 */
class SyncWorker(
    context: Context,
    params: WorkerParameters,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val app = applicationContext as? KasKuApplication ?: return Result.failure()
        val engine = app.container.syncEngine
        return try {
            engine.syncAll()
            Result.success()
        } catch (e: Exception) {
            // HTTP/network gagal → biarkan WorkManager retry (queue tetap PENDING/FAILED).
            Result.retry()
        }
    }

    companion object {
        private const val UNIQUE_ONESHOT = "kasku-sync-oneshot"
        private const val UNIQUE_PERIODIC = "kasku-sync-periodic"

        private val networkConstraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        /** Sync sekali secepatnya (dipakai saat masuk Home / setelah mutasi). */
        fun enqueueOnce(context: Context) {
            val req = OneTimeWorkRequestBuilder<SyncWorker>()
                .setConstraints(networkConstraints)
                .build()
            WorkManager.getInstance(context)
                .enqueueUniqueWork(UNIQUE_ONESHOT, ExistingWorkPolicy.KEEP, req)
        }

        /** Sync periodik 15m (minimum WorkManager). Golden ref 5m tak mungkin di WM. */
        fun enqueuePeriodic(context: Context) {
            val req = PeriodicWorkRequestBuilder<SyncWorker>(15, TimeUnit.MINUTES)
                .setConstraints(networkConstraints)
                .build()
            WorkManager.getInstance(context)
                .enqueueUniquePeriodicWork(UNIQUE_PERIODIC, ExistingPeriodicWorkPolicy.KEEP, req)
        }
    }
}
