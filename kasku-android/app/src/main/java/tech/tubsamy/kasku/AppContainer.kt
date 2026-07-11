package tech.tubsamy.kasku

import android.content.Context
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.AuthRepository
import tech.tubsamy.kasku.data.TokenStore
import tech.tubsamy.kasku.data.remote.AuthApi
import tech.tubsamy.kasku.data.remote.FinanceApi
import tech.tubsamy.kasku.data.remote.Network

/**
 * DI manual. ponytail: cukup untuk sekarang — ganti ke Hilt saat graph mulai
 * menyakitkan (banyak scope/ViewModel).
 */
class AppContainer(context: Context) {
    val tokenStore = TokenStore(context.applicationContext)
    private val retrofit = Network.retrofit(tokenStore)
    private val authApi = retrofit.create(AuthApi::class.java)
    private val financeApi = retrofit.create(FinanceApi::class.java)
    val authRepository = AuthRepository(authApi, tokenStore, Network.json)
    val accountsRepository = AccountsRepository(financeApi, Network.json)
}
