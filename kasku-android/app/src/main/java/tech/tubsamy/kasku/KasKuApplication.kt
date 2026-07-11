package tech.tubsamy.kasku

import android.app.Application
import kotlinx.coroutines.runBlocking

class KasKuApplication : Application() {
    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        container = AppContainer(this)
        // Muat token tersimpan agar startDestination bisa langsung tahu status login.
        runBlocking { container.tokenStore.warmCache() }
    }
}
