package tech.tubsamy.kasku

import android.content.Context
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.AuthRepository
import tech.tubsamy.kasku.data.TokenStore
import tech.tubsamy.kasku.data.remote.Network

/**
 * DI manual. ponytail: cukup untuk sekarang — ganti ke Hilt saat graph mulai
 * menyakitkan (banyak scope/ViewModel).
 */
class AppContainer(context: Context) {
    val tokenStore = TokenStore(context.applicationContext)
    private val apis = Network.build(context, tokenStore)
    val authRepository = AuthRepository(apis.authApi, tokenStore, apis.cookieJar, Network.json)
    val accountsRepository = AccountsRepository(apis.financeApi, Network.json)
}
