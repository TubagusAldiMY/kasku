package tech.tubsamy.kasku.data.remote

import android.content.Context
import okhttp3.Cookie
import okhttp3.CookieJar
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

/**
 * CookieJar sederhana yang mem-persist cookie (khususnya `refresh_token` 30 hari)
 * ke SharedPreferences, agar sesi bertahan lintas restart aplikasi.
 *
 * ponytail: SharedPreferences polos untuk sekarang — refresh token itu sensitif;
 * upgrade ke EncryptedSharedPreferences/Keystore saat hardening keamanan.
 */
class PersistentCookieJar(context: Context) : CookieJar {

    private val prefs = context.getSharedPreferences("kasku_cookies", Context.MODE_PRIVATE)

    // host -> (nama cookie -> Cookie)
    private val cache = mutableMapOf<String, MutableMap<String, Cookie>>()

    init {
        prefs.all.forEach { (host, serialized) ->
            val url = "https://$host".toHttpUrlOrNull() ?: return@forEach
            (serialized as? String)?.split("\n")?.forEach { line ->
                Cookie.parse(url, line)?.let { c ->
                    cache.getOrPut(host) { mutableMapOf() }[c.name] = c
                }
            }
        }
    }

    @Synchronized
    override fun saveFromResponse(url: HttpUrl, cookies: List<Cookie>) {
        val map = cache.getOrPut(url.host) { mutableMapOf() }
        cookies.forEach { map[it.name] = it }
        persist(url.host)
    }

    @Synchronized
    override fun loadForRequest(url: HttpUrl): List<Cookie> {
        val now = System.currentTimeMillis()
        val map = cache[url.host] ?: return emptyList()
        val expired = map.filterValues { it.expiresAt <= now }.keys
        if (expired.isNotEmpty()) {
            expired.forEach { map.remove(it) }
            persist(url.host)
        }
        return map.values.filter { it.matches(url) }
    }

    @Synchronized
    fun clear() {
        cache.clear()
        prefs.edit().clear().apply()
    }

    private fun persist(host: String) {
        val serialized = cache[host]?.values?.joinToString("\n") { it.toString() }.orEmpty()
        prefs.edit().putString(host, serialized).apply()
    }
}
