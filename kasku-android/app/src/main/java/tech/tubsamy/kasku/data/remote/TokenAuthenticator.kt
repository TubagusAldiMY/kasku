package tech.tubsamy.kasku.data.remote

import kotlinx.coroutines.runBlocking
import okhttp3.Authenticator
import okhttp3.Request
import okhttp3.Response
import okhttp3.Route
import tech.tubsamy.kasku.data.TokenCache

/**
 * Saat request ber-auth kena 401 (access token 15m kadaluwarsa), panggil /auth/refresh
 * (refresh token dibawa cookie), simpan access token baru, lalu ulangi request asli.
 * Gagal refresh (token 30d habis / reuse terdeteksi) → bersihkan sesi → menyerah (null),
 * biar UI menampilkan "sesi berakhir".
 */
class TokenAuthenticator(
    private val tokenStore: TokenCache,
    private val refreshApi: AuthApi,
) : Authenticator {

    override fun authenticate(route: Route?, response: Response): Request? {
        // Hanya tangani request yang memang mengirim Bearer (bukan login/refresh).
        val sentToken = response.request.header("Authorization")?.removePrefix("Bearer ") ?: return null

        // Cegah loop: kalau sudah pernah retry di rantai ini, menyerah.
        if (responseCount(response) >= 2) return null

        synchronized(this) {
            val current = tokenStore.cachedAccessToken()
            // Thread lain mungkin sudah refresh — pakai token baru tanpa refresh ulang.
            if (current != null && current != sentToken) {
                return response.request.retryWith(current)
            }

            val newToken = runBlocking {
                runCatching { refreshApi.refresh().data?.access_token }.getOrNull()
            }

            if (newToken.isNullOrBlank()) {
                runBlocking { tokenStore.clear() }
                return null
            }

            runBlocking { tokenStore.saveAccessToken(newToken) }
            return response.request.retryWith(newToken)
        }
    }

    private fun Request.retryWith(token: String): Request =
        newBuilder().header("Authorization", "Bearer $token").build()

    private fun responseCount(response: Response): Int {
        var r: Response? = response
        var count = 1
        while (r?.priorResponse != null) {
            count++
            r = r.priorResponse
        }
        return count
    }
}
