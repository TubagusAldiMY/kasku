package tech.tubsamy.kasku

import android.content.Context
import tech.tubsamy.kasku.data.AccountsRepository
import tech.tubsamy.kasku.data.AuthRepository
import tech.tubsamy.kasku.data.BillingRepository
import tech.tubsamy.kasku.data.BudgetsRepository
import tech.tubsamy.kasku.data.CategoriesRepository
import tech.tubsamy.kasku.data.ConflictsRepository
import tech.tubsamy.kasku.data.DebtsRepository
import tech.tubsamy.kasku.data.InvestmentMarketRepository
import tech.tubsamy.kasku.data.InvestmentsRepository
import tech.tubsamy.kasku.data.ProfileRepository
import tech.tubsamy.kasku.data.ReportsRepository
import tech.tubsamy.kasku.data.TokenStore
import tech.tubsamy.kasku.data.TransactionsRepository
import tech.tubsamy.kasku.data.local.KasKuDatabase
import tech.tubsamy.kasku.data.remote.Network
import tech.tubsamy.kasku.data.sync.AccountMutations
import tech.tubsamy.kasku.data.sync.ConflictResolutionService
import tech.tubsamy.kasku.data.sync.InvestmentMutations
import tech.tubsamy.kasku.data.sync.RoomSyncStore
import tech.tubsamy.kasku.data.sync.SyncEngine
import tech.tubsamy.kasku.data.sync.SyncWorker
import tech.tubsamy.kasku.data.sync.TransactionMutations

/**
 * DI manual. ponytail: cukup untuk sekarang — ganti ke Hilt saat graph mulai
 * menyakitkan (banyak scope/ViewModel).
 */
class AppContainer(context: Context) {
    private val appContext = context.applicationContext

    val tokenStore = TokenStore(appContext)
    private val apis = Network.build(appContext, tokenStore)
    val authRepository = AuthRepository(apis.authApi, tokenStore, apis.cookieJar, Network.json)

    // Offline-first (M2)
    private val db = KasKuDatabase.build(appContext)
    private val syncStore = RoomSyncStore(db)
    val syncEngine = SyncEngine(syncStore, apis.syncApi, Network.json)
    val accountsRepository = AccountsRepository(db)
    val accountMutations = AccountMutations(
        db = db,
        json = Network.json,
        fireSync = { SyncWorker.enqueueOnce(appContext) },
    )
    val transactionMutations = TransactionMutations(
        db = db,
        json = Network.json,
        fireSync = { SyncWorker.enqueueOnce(appContext) },
    )

    // Slice C — data layer 4 fitur (Dashboard / Riwayat / Kategori / Konflik).
    // Satu instance CategoriesRepository di-share (satu cache in-memory).
    val categoriesRepository = CategoriesRepository(apis.financeApi)
    val budgetsRepository = BudgetsRepository(apis.financeApi)
    val transactionsRepository = TransactionsRepository(db, categoriesRepository)
    val conflictsRepository = ConflictsRepository(db)
    val conflictResolutionService = ConflictResolutionService(
        db = db,
        conflicts = conflictsRepository,
        accountMutations = accountMutations,
        transactionMutations = transactionMutations,
        json = Network.json,
    )

    // Investasi (offline-first daftar instrumen; harga live + riwayat + catat txn = online REST)
    val investmentsRepository = InvestmentsRepository(db)
    val investmentMarketRepository = InvestmentMarketRepository(apis.priceApi, apis.investmentApi)
    val investmentMutations = InvestmentMutations(
        db = db,
        json = Network.json,
        fireSync = { SyncWorker.enqueueOnce(appContext) },
    )

    // Hutang & Piutang / Laporan / Profil / Langganan — read-oriented, online-only.
    val debtsRepository = DebtsRepository(apis.debtApi)
    val reportsRepository = ReportsRepository(apis.reportApi)
    val profileRepository = ProfileRepository(apis.userApi)
    val billingRepository = BillingRepository(apis.billingApi)
}
