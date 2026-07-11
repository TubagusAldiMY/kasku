package tech.tubsamy.kasku

import android.app.Application
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import tech.tubsamy.kasku.data.sync.SyncWorker

class KasKuApplication : Application() {
    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        container = AppContainer(this)
        // Muat token tersimpan agar startDestination bisa langsung tahu status login.
        runBlocking { container.tokenStore.warmCache() }

        // Offline-first init: pulihkan operasi IN_FLIGHT yatim (crash saat push) →
        // PENDING, lalu jadwalkan sync periodik. one-off saat login/masuk Home.
        CoroutineScope(Dispatchers.IO).launch {
            container.syncEngine.recoverStuck()
        }
        SyncWorker.enqueuePeriodic(this)
        if (container.authRepository.isLoggedIn()) {
            SyncWorker.enqueueOnce(this)
        }
    }
}
