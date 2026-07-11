package tech.tubsamy.kasku.data

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map

private val Context.dataStore by preferencesDataStore(name = "kasku_auth")

/** Kontrak akses access token — memisahkan TokenAuthenticator dari detail Android (mudah di-test). */
interface TokenCache {
    fun cachedAccessToken(): String?
    suspend fun saveAccessToken(token: String)
    suspend fun clear()
}

/**
 * Menyimpan access token. Persist di DataStore + cache in-memory agar interceptor
 * OkHttp bisa membacanya sinkron.
 *
 * ponytail: DataStore polos untuk sekarang; enkripsi Keystore + refresh token
 * ditambah saat alur refresh (M1.5) dibangun — access token TTL 15m, risiko rendah.
 */
class TokenStore(private val context: Context) : TokenCache {

    @Volatile
    private var cached: String? = null

    private val key = stringPreferencesKey("access_token")

    /** Muat token tersimpan ke cache saat startup. */
    suspend fun warmCache() {
        cached = context.dataStore.data.map { it[key] }.first()
    }

    override fun cachedAccessToken(): String? = cached

    override suspend fun saveAccessToken(token: String) {
        cached = token
        context.dataStore.edit { it[key] = token }
    }

    override suspend fun clear() {
        cached = null
        context.dataStore.edit { it.remove(key) }
    }
}
