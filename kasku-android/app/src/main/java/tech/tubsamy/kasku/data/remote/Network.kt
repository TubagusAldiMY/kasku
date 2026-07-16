package tech.tubsamy.kasku.data.remote

import android.content.Context
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import tech.tubsamy.kasku.BuildConfig
import tech.tubsamy.kasku.data.TokenStore

/** Kumpulan API + CookieJar hasil perakitan jaringan. */
class NetworkApis(
    val authApi: AuthApi,
    val financeApi: FinanceApi,
    val syncApi: SyncApi,
    val debtApi: DebtApi,
    val reportApi: ReportApi,
    val userApi: UserApi,
    val billingApi: BillingApi,
    val priceApi: PriceApi,
    val investmentApi: InvestmentApi,
    val cookieJar: PersistentCookieJar,
)

object Network {

    val json: Json = Json {
        ignoreUnknownKeys = true
        encodeDefaults = true
    }

    fun build(context: Context, tokenStore: TokenStore): NetworkApis {
        val cookieJar = PersistentCookieJar(context.applicationContext)
        val converter = json.asConverterFactory("application/json".toMediaType())

        fun logging() = HttpLoggingInterceptor().apply {
            level = if (BuildConfig.DEBUG) HttpLoggingInterceptor.Level.BASIC else HttpLoggingInterceptor.Level.NONE
        }

        // Client khusus refresh: bawa cookie, TANPA Authenticator (cegah rekursi).
        val refreshClient = OkHttpClient.Builder()
            .cookieJar(cookieJar)
            .addInterceptor(logging())
            .build()
        val refreshApi = Retrofit.Builder()
            .baseUrl(BuildConfig.BASE_URL)
            .client(refreshClient)
            .addConverterFactory(converter)
            .build()
            .create(AuthApi::class.java)

        val authenticator = TokenAuthenticator(tokenStore, refreshApi)

        // Client utama: sisip Bearer + auto-refresh saat 401 + simpan cookie.
        val mainClient = OkHttpClient.Builder()
            .cookieJar(cookieJar)
            .addInterceptor { chain ->
                val token = tokenStore.cachedAccessToken()
                val request = if (token != null) {
                    chain.request().newBuilder().addHeader("Authorization", "Bearer $token").build()
                } else {
                    chain.request()
                }
                chain.proceed(request)
            }
            .addInterceptor(logging())
            .authenticator(authenticator)
            .build()

        val mainRetrofit = Retrofit.Builder()
            .baseUrl(BuildConfig.BASE_URL)
            .client(mainClient)
            .addConverterFactory(converter)
            .build()

        return NetworkApis(
            authApi = mainRetrofit.create(AuthApi::class.java),
            financeApi = mainRetrofit.create(FinanceApi::class.java),
            syncApi = mainRetrofit.create(SyncApi::class.java),
            debtApi = mainRetrofit.create(DebtApi::class.java),
            reportApi = mainRetrofit.create(ReportApi::class.java),
            userApi = mainRetrofit.create(UserApi::class.java),
            billingApi = mainRetrofit.create(BillingApi::class.java),
            priceApi = mainRetrofit.create(PriceApi::class.java),
            investmentApi = mainRetrofit.create(InvestmentApi::class.java),
            cookieJar = cookieJar,
        )
    }
}
