package tech.tubsamy.kasku

import android.content.Context
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.AuthRepository
import tech.tubsamy.kasku.data.BillingRepository
import tech.tubsamy.kasku.data.BudgetsRepository
import tech.tubsamy.kasku.data.CategoriesRepository
import tech.tubsamy.kasku.data.DebtsRepository
import tech.tubsamy.kasku.data.InvestmentMarketRepository
import tech.tubsamy.kasku.data.InvestmentsRepository
import tech.tubsamy.kasku.data.ProfileRepository
import tech.tubsamy.kasku.data.ReportsRepository
import tech.tubsamy.kasku.data.TokenStore
import tech.tubsamy.kasku.data.TransactionsRepository
import tech.tubsamy.kasku.data.remote.Network

/**
 * DI manual. ponytail: cukup untuk sekarang — ganti ke Hilt saat graph mulai
 * menyakitkan (banyak scope/ViewModel).
 */
class AppContainer(context: Context) {
    private val appContext = context.applicationContext

    val tokenStore = TokenStore(appContext)
    private val apis = Network.build(appContext, tokenStore)
    val authRepository = AuthRepository(apis.authApi, tokenStore, apis.cookieJar, Network.json)

    // Semua repo ONLINE (cache in-memory StateFlow, mengikuti pola BudgetsRepository).
    // Satu instance CategoriesRepository di-share (satu cache in-memory).
    val categoriesRepository = CategoriesRepository(apis.financeApi)
    val budgetsRepository = BudgetsRepository(apis.financeApi)
    val accountsRepository = AccountsRepository(apis.financeApi)
    val transactionsRepository = TransactionsRepository(apis.transactionApi, categoriesRepository)

    // Investasi: daftar instrumen (REST) + harga live/riwayat/pencatatan unit (REST).
    val investmentsRepository = InvestmentsRepository(apis.investmentApi)
    val investmentMarketRepository = InvestmentMarketRepository(apis.priceApi, apis.investmentApi)

    // Hutang & Piutang / Laporan / Profil / Langganan — read-oriented, online.
    val debtsRepository = DebtsRepository(apis.debtApi)
    val reportsRepository = ReportsRepository(apis.reportApi)
    val profileRepository = ProfileRepository(apis.userApi)
    val billingRepository = BillingRepository(apis.billingApi)
}
