package tech.tubsamy.kasku.data

import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import tech.tubsamy.kasku.data.remote.AuthApi
import tech.tubsamy.kasku.data.remote.TokenAuthenticator

class TokenAuthenticatorTest {

    private lateinit var server: MockWebServer
    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

    // Fake in-memory cache — tanpa Android Context.
    private class FakeCache(initial: String?) : TokenCache {
        var token: String? = initial
        var cleared = false
        override fun cachedAccessToken() = token
        override suspend fun saveAccessToken(token: String) { this.token = token }
        override suspend fun clear() { token = null; cleared = true }
    }

    @Before fun setUp() { server = MockWebServer(); server.start() }
    @After fun tearDown() { server.shutdown() }

    private fun retrofit(client: OkHttpClient) = Retrofit.Builder()
        .baseUrl(server.url("/"))
        .client(client)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()

    @Test
    fun refreshesOn401ThenRetriesWithNewToken() {
        // 1) request terproteksi → 401. 2) /auth/refresh → token baru. 3) retry → 200.
        server.enqueue(MockResponse().setResponseCode(401))
        server.enqueue(
            MockResponse().setResponseCode(200)
                .setBody("""{"success":true,"data":{"access_token":"NEW_TOKEN","token_type":"Bearer","expires_in":900}}"""),
        )
        server.enqueue(MockResponse().setResponseCode(200).setBody("ok"))

        val cache = FakeCache("OLD_TOKEN")
        val refreshApi = retrofit(OkHttpClient()).create(AuthApi::class.java)

        val client = OkHttpClient.Builder()
            .addInterceptor { chain ->
                val t = cache.cachedAccessToken()
                val req = if (t != null) chain.request().newBuilder().addHeader("Authorization", "Bearer $t").build() else chain.request()
                chain.proceed(req)
            }
            .authenticator(TokenAuthenticator(cache, refreshApi))
            .build()

        val resp = client.newCall(Request.Builder().url(server.url("/accounts")).build()).execute()

        assertEquals(200, resp.code)
        assertEquals("NEW_TOKEN", cache.token)
        // 3 request: /accounts(401) → /auth/refresh → /accounts(retry)
        assertEquals(3, server.requestCount)
        resp.close()
    }

    @Test
    fun clearsSessionWhenRefreshFails() {
        server.enqueue(MockResponse().setResponseCode(401)) // request terproteksi
        server.enqueue(MockResponse().setResponseCode(401)) // refresh juga gagal

        val cache = FakeCache("OLD_TOKEN")
        val refreshApi = retrofit(OkHttpClient()).create(AuthApi::class.java)

        val client = OkHttpClient.Builder()
            .addInterceptor { chain ->
                chain.proceed(chain.request().newBuilder().addHeader("Authorization", "Bearer ${cache.cachedAccessToken()}").build())
            }
            .authenticator(TokenAuthenticator(cache, refreshApi))
            .build()

        val resp = client.newCall(Request.Builder().url(server.url("/accounts")).build()).execute()

        assertEquals(401, resp.code)
        assertEquals(true, cache.cleared)
        resp.close()
    }
}
