package tech.tubsamy.kasku

import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.outlined.AccountBalanceWallet
import androidx.compose.material.icons.outlined.Category
import androidx.compose.material.icons.outlined.Dashboard
import androidx.compose.material.icons.outlined.ReceiptLong
import androidx.compose.material.icons.outlined.TrendingUp
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavHostController
import androidx.navigation.compose.currentBackStackEntryAsState
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.ui.account.AddAccountScreen
import tech.tubsamy.kasku.ui.account.AddAccountViewModel
import tech.tubsamy.kasku.ui.auth.ForgotPasswordScreen
import tech.tubsamy.kasku.ui.auth.ForgotPasswordViewModel
import tech.tubsamy.kasku.ui.auth.LoginScreen
import tech.tubsamy.kasku.ui.auth.LoginViewModel
import tech.tubsamy.kasku.ui.auth.RegisterScreen
import tech.tubsamy.kasku.ui.auth.RegisterViewModel
import tech.tubsamy.kasku.ui.auth.ResetPasswordScreen
import tech.tubsamy.kasku.ui.auth.ResetPasswordViewModel
import tech.tubsamy.kasku.ui.auth.VerifyEmailScreen
import tech.tubsamy.kasku.ui.auth.VerifyEmailViewModel
import tech.tubsamy.kasku.ui.billing.BillingScreen
import tech.tubsamy.kasku.ui.billing.BillingViewModel
import tech.tubsamy.kasku.ui.budget.AddBudgetScreen
import tech.tubsamy.kasku.ui.budget.AddBudgetViewModel
import tech.tubsamy.kasku.ui.budget.BudgetScreen
import tech.tubsamy.kasku.ui.budget.BudgetViewModel
import tech.tubsamy.kasku.ui.category.CategoryScreen
import tech.tubsamy.kasku.ui.category.CategoryViewModel
import tech.tubsamy.kasku.ui.conflicts.ConflictsScreen
import tech.tubsamy.kasku.ui.conflicts.ConflictsViewModel
import tech.tubsamy.kasku.ui.dashboard.DashboardScreen
import tech.tubsamy.kasku.ui.dashboard.DashboardViewModel
import tech.tubsamy.kasku.ui.debt.AddDebtScreen
import tech.tubsamy.kasku.ui.debt.AddDebtViewModel
import tech.tubsamy.kasku.ui.debt.DebtPaymentsScreen
import tech.tubsamy.kasku.ui.debt.DebtPaymentsViewModel
import tech.tubsamy.kasku.ui.debt.DebtsScreen
import tech.tubsamy.kasku.ui.debt.DebtViewModel
import tech.tubsamy.kasku.ui.home.HomeScreen
import tech.tubsamy.kasku.ui.home.HomeViewModel
import tech.tubsamy.kasku.ui.investment.AddInvestmentScreen
import tech.tubsamy.kasku.ui.investment.AddInvestmentViewModel
import tech.tubsamy.kasku.ui.investment.InvestmentScreen
import tech.tubsamy.kasku.ui.investment.InvestmentViewModel
import tech.tubsamy.kasku.ui.profile.ChangePasswordScreen
import tech.tubsamy.kasku.ui.profile.ChangePasswordViewModel
import tech.tubsamy.kasku.ui.profile.ProfileScreen
import tech.tubsamy.kasku.ui.profile.ProfileViewModel
import tech.tubsamy.kasku.ui.report.ReportsScreen
import tech.tubsamy.kasku.ui.report.ReportsViewModel
import tech.tubsamy.kasku.ui.transaction.AddTransactionScreen
import tech.tubsamy.kasku.ui.transaction.AddTransactionViewModel
import tech.tubsamy.kasku.ui.transaction.TransactionHistoryScreen
import tech.tubsamy.kasku.ui.transaction.TransactionHistoryViewModel

private object Routes {
    const val LOGIN = "login"
    const val REGISTER = "register"
    const val HOME = "home"
    const val ADD_TRANSACTION = "add_transaction"
    const val EDIT_TRANSACTION = "edit_transaction" // arg: transactionId
    const val DASHBOARD = "dashboard"
    const val HISTORY = "history"
    const val CONFLICTS = "conflicts"
    const val CATEGORIES = "categories"
    const val INVESTMENTS = "investments"
    const val ADD_INVESTMENT = "add_investment"
    const val BUDGETS = "budgets"
    const val ADD_BUDGET = "add_budget"
    const val EDIT_BUDGET = "edit_budget" // arg: budgetId
    const val ADD_ACCOUNT = "add_account"
    const val EDIT_ACCOUNT = "edit_account" // arg: accountId
    const val FORGOT_PASSWORD = "forgot_password"
    const val RESET_PASSWORD = "reset_password"
    const val VERIFY_EMAIL = "verify_email"
    const val DEBTS = "debts"
    const val ADD_DEBT = "add_debt"
    const val EDIT_DEBT = "edit_debt" // arg: debtId
    const val DEBT_PAYMENTS = "debt_payments" // arg: debtId
    const val REPORTS = "reports"
    const val PROFILE = "profile"
    const val CHANGE_PASSWORD = "change_password"
    const val BILLING = "billing"
}

@Composable
fun AppRoot(container: AppContainer) {
    val nav = rememberNavController()
    val scope = rememberCoroutineScope()
    val appContext = LocalContext.current.applicationContext
    // Landing = Dashboard (ringkasan) saat sudah login.
    val start = if (container.authRepository.isLoggedIn()) Routes.DASHBOARD else Routes.LOGIN

    val backStack by nav.currentBackStackEntryAsState()
    val currentRoute = backStack?.destination?.route

    Scaffold(
        bottomBar = {
            if (currentRoute in TAB_ROUTES) KasKuBottomBar(nav = nav, currentRoute = currentRoute)
        },
        floatingActionButton = {
            // FAB tambah transaksi tersedia di Dashboard & Akun (layar berorientasi transaksi).
            if (currentRoute == Routes.DASHBOARD || currentRoute == Routes.HOME) {
                FloatingActionButton(onClick = { nav.navigate(Routes.ADD_TRANSACTION) }) {
                    Icon(Icons.Filled.Add, contentDescription = "Tambah transaksi")
                }
            }
        },
    ) { innerPadding ->
        NavHost(
            navController = nav,
            startDestination = start,
            modifier = Modifier.padding(innerPadding),
        ) {
        composable(Routes.LOGIN) {
            val vm: LoginViewModel = viewModel(factory = LoginViewModel.factory(container.authRepository))
            LoginScreen(
                vm = vm,
                onLoggedIn = {
                    nav.navigate(Routes.DASHBOARD) {
                        popUpTo(Routes.LOGIN) { inclusive = true }
                    }
                },
                onRegister = { nav.navigate(Routes.REGISTER) },
                onForgotPassword = { nav.navigate(Routes.FORGOT_PASSWORD) },
            )
        }
        composable(Routes.REGISTER) {
            val vm: RegisterViewModel = viewModel(factory = RegisterViewModel.factory(container.authRepository))
            RegisterScreen(
                vm = vm,
                onBackToLogin = { nav.popBackStack(Routes.LOGIN, inclusive = false) },
            )
        }
        composable(Routes.FORGOT_PASSWORD) {
            val vm: ForgotPasswordViewModel = viewModel(
                factory = ForgotPasswordViewModel.factory(container.authRepository),
            )
            ForgotPasswordScreen(
                vm = vm,
                onBack = { nav.popBackStack() },
                onHaveCode = { nav.navigate(Routes.RESET_PASSWORD) },
                onVerifyEmail = { nav.navigate(Routes.VERIFY_EMAIL) },
            )
        }
        composable(Routes.RESET_PASSWORD) {
            val vm: ResetPasswordViewModel = viewModel(
                factory = ResetPasswordViewModel.factory(container.authRepository),
            )
            ResetPasswordScreen(
                vm = vm,
                onDone = { nav.popBackStack(Routes.LOGIN, inclusive = false) },
                onBack = { nav.popBackStack() },
            )
        }
        composable(Routes.VERIFY_EMAIL) {
            val vm: VerifyEmailViewModel = viewModel(
                factory = VerifyEmailViewModel.factory(container.authRepository),
            )
            VerifyEmailScreen(
                vm = vm,
                onDone = { nav.popBackStack(Routes.LOGIN, inclusive = false) },
                onBack = { nav.popBackStack() },
            )
        }
        composable(Routes.HOME) {
            val vm: HomeViewModel = viewModel(
                factory = HomeViewModel.factory(
                    container.accountsRepository,
                    container.conflictsRepository,
                    container.accountMutations,
                    appContext,
                ),
            )
            HomeScreen(
                vm = vm,
                onAddAccount = { nav.navigate(Routes.ADD_ACCOUNT) },
                onEditAccount = { id -> nav.navigate("${Routes.EDIT_ACCOUNT}/$id") },
                onConflicts = { nav.navigate(Routes.CONFLICTS) },
                onDebts = { nav.navigate(Routes.DEBTS) },
                onReports = { nav.navigate(Routes.REPORTS) },
                onBilling = { nav.navigate(Routes.BILLING) },
                onProfile = { nav.navigate(Routes.PROFILE) },
                onLogout = {
                    scope.launch {
                        container.authRepository.logout()
                        nav.navigate(Routes.LOGIN) {
                            popUpTo(nav.graph.findStartDestination().id) { inclusive = true }
                        }
                    }
                },
            )
        }
        composable(Routes.ADD_ACCOUNT) {
            val vm: AddAccountViewModel = viewModel(
                factory = AddAccountViewModel.factory(container.accountMutations, container.accountsRepository),
            )
            AddAccountScreen(
                vm = vm,
                onSaved = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
        composable(
            route = "${Routes.EDIT_ACCOUNT}/{accountId}",
            arguments = listOf(navArgument("accountId") { type = NavType.StringType }),
        ) { backStackEntry ->
            val accountId = backStackEntry.arguments?.getString("accountId")
            if (accountId == null) {
                nav.popBackStack()
                return@composable
            }
            val vm: AddAccountViewModel = viewModel(
                factory = AddAccountViewModel.factory(
                    container.accountMutations,
                    container.accountsRepository,
                    accountId,
                ),
            )
            AddAccountScreen(
                vm = vm,
                onSaved = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
        composable(Routes.DEBTS) {
            val vm: DebtViewModel = viewModel(
                factory = DebtViewModel.factory(container.debtsRepository),
            )
            DebtsScreen(
                vm = vm,
                onBack = { nav.popBackStack() },
                onAddDebt = { nav.navigate(Routes.ADD_DEBT) },
                onEditDebt = { id -> nav.navigate("${Routes.EDIT_DEBT}/$id") },
                onOpenPayments = { id -> nav.navigate("${Routes.DEBT_PAYMENTS}/$id") },
            )
        }
        composable(Routes.ADD_DEBT) {
            val vm: AddDebtViewModel = viewModel(
                factory = AddDebtViewModel.factory(container.debtsRepository),
            )
            AddDebtScreen(
                vm = vm,
                onSaved = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
        composable(
            route = "${Routes.EDIT_DEBT}/{debtId}",
            arguments = listOf(navArgument("debtId") { type = NavType.StringType }),
        ) { backStackEntry ->
            val debtId = backStackEntry.arguments?.getString("debtId")
            if (debtId == null) {
                nav.popBackStack()
                return@composable
            }
            val vm: AddDebtViewModel = viewModel(
                factory = AddDebtViewModel.factory(container.debtsRepository, debtId),
            )
            AddDebtScreen(
                vm = vm,
                onSaved = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
        composable(
            route = "${Routes.DEBT_PAYMENTS}/{debtId}",
            arguments = listOf(navArgument("debtId") { type = NavType.StringType }),
        ) { backStackEntry ->
            val debtId = backStackEntry.arguments?.getString("debtId")
            if (debtId == null) {
                nav.popBackStack()
                return@composable
            }
            val vm: DebtPaymentsViewModel = viewModel(
                factory = DebtPaymentsViewModel.factory(container.debtsRepository, debtId),
            )
            DebtPaymentsScreen(vm = vm, onBack = { nav.popBackStack() })
        }
        composable(Routes.REPORTS) {
            val vm: ReportsViewModel = viewModel(
                factory = ReportsViewModel.factory(container.reportsRepository),
            )
            ReportsScreen(vm = vm, onBack = { nav.popBackStack() })
        }
        composable(Routes.PROFILE) {
            val vm: ProfileViewModel = viewModel(
                factory = ProfileViewModel.factory(container.profileRepository),
            )
            ProfileScreen(
                vm = vm,
                onChangePassword = { nav.navigate(Routes.CHANGE_PASSWORD) },
                onLogout = {
                    scope.launch {
                        container.authRepository.logout()
                        nav.navigate(Routes.LOGIN) {
                            popUpTo(nav.graph.findStartDestination().id) { inclusive = true }
                        }
                    }
                },
            )
        }
        composable(Routes.CHANGE_PASSWORD) {
            val vm: ChangePasswordViewModel = viewModel(
                factory = ChangePasswordViewModel.factory(container.authRepository),
            )
            ChangePasswordScreen(
                vm = vm,
                onDone = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
        composable(Routes.BILLING) {
            val vm: BillingViewModel = viewModel(
                factory = BillingViewModel.factory(container.billingRepository),
            )
            BillingScreen(vm = vm, onBack = { nav.popBackStack() })
        }
        composable(Routes.ADD_TRANSACTION) {
            val vm: AddTransactionViewModel = viewModel(
                factory = AddTransactionViewModel.factory(
                    container.accountsRepository,
                    container.transactionMutations,
                    container.categoriesRepository,
                    container.transactionsRepository,
                ),
            )
            AddTransactionScreen(
                vm = vm,
                onSaved = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
        composable(
            route = "${Routes.EDIT_TRANSACTION}/{transactionId}",
            arguments = listOf(navArgument("transactionId") { type = NavType.StringType }),
        ) { backStackEntry ->
            val transactionId = backStackEntry.arguments?.getString("transactionId")
            if (transactionId == null) {
                nav.popBackStack()
                return@composable
            }
            val vm: AddTransactionViewModel = viewModel(
                factory = AddTransactionViewModel.factory(
                    container.accountsRepository,
                    container.transactionMutations,
                    container.categoriesRepository,
                    container.transactionsRepository,
                    transactionId,
                ),
            )
            AddTransactionScreen(
                vm = vm,
                onSaved = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
        composable(Routes.DASHBOARD) {
            val vm: DashboardViewModel = viewModel(
                factory = DashboardViewModel.factory(
                    container.accountsRepository,
                    container.transactionsRepository,
                    container.categoriesRepository,
                    container.budgetsRepository,
                    appContext,
                ),
            )
            DashboardScreen(
                vm = vm,
                onBack = { nav.popBackStack() },
                onSeeAllHistory = {
                    // Sama seperti pindah tab: satu instance, state dipulihkan.
                    nav.navigate(Routes.HISTORY) {
                        popUpTo(nav.graph.findStartDestination().id) { saveState = true }
                        launchSingleTop = true
                        restoreState = true
                    }
                },
                onManageBudgets = { nav.navigate(Routes.BUDGETS) },
            )
        }
        composable(Routes.HISTORY) {
            val vm: TransactionHistoryViewModel = viewModel(
                factory = TransactionHistoryViewModel.factory(
                    container.transactionsRepository,
                    container.categoriesRepository,
                    container.transactionMutations,
                ),
            )
            TransactionHistoryScreen(
                vm = vm,
                onBack = { nav.popBackStack() },
                onEdit = { id -> nav.navigate("${Routes.EDIT_TRANSACTION}/$id") },
            )
        }
        composable(Routes.CONFLICTS) {
            val vm: ConflictsViewModel = viewModel(
                factory = ConflictsViewModel.factory(
                    container.conflictsRepository,
                    container.conflictResolutionService,
                ),
            )
            ConflictsScreen(vm = vm, onBack = { nav.popBackStack() })
        }
        composable(Routes.CATEGORIES) {
            val vm: CategoryViewModel = viewModel(
                factory = CategoryViewModel.factory(container.categoriesRepository),
            )
            CategoryScreen(vm = vm, onBack = { nav.popBackStack() })
        }
        composable(Routes.INVESTMENTS) {
            val vm: InvestmentViewModel = viewModel(
                factory = InvestmentViewModel.factory(
                    container.investmentsRepository,
                    container.investmentMutations,
                    appContext,
                ),
            )
            InvestmentScreen(
                vm = vm,
                onBack = { nav.popBackStack() },
                onAddInvestment = { nav.navigate(Routes.ADD_INVESTMENT) },
            )
        }
        composable(Routes.BUDGETS) {
            val vm: BudgetViewModel = viewModel(
                factory = BudgetViewModel.factory(container.budgetsRepository),
            )
            BudgetScreen(
                vm = vm,
                onBack = { nav.popBackStack() },
                onAddBudget = { nav.navigate(Routes.ADD_BUDGET) },
                onEditBudget = { id -> nav.navigate("${Routes.EDIT_BUDGET}/$id") },
            )
        }
        composable(Routes.ADD_BUDGET) {
            val vm: AddBudgetViewModel = viewModel(
                factory = AddBudgetViewModel.factory(container.budgetsRepository, container.categoriesRepository),
            )
            AddBudgetScreen(
                vm = vm,
                onSaved = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
        composable(
            route = "${Routes.EDIT_BUDGET}/{budgetId}",
            arguments = listOf(navArgument("budgetId") { type = NavType.StringType }),
        ) { backStackEntry ->
            val budgetId = backStackEntry.arguments?.getString("budgetId")
            if (budgetId == null) {
                nav.popBackStack()
                return@composable
            }
            val vm: AddBudgetViewModel = viewModel(
                factory = AddBudgetViewModel.factory(
                    container.budgetsRepository,
                    container.categoriesRepository,
                    budgetId,
                ),
            )
            AddBudgetScreen(
                vm = vm,
                onSaved = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
        composable(Routes.ADD_INVESTMENT) {
            val vm: AddInvestmentViewModel = viewModel(
                factory = AddInvestmentViewModel.factory(container.investmentMutations),
            )
            AddInvestmentScreen(
                vm = vm,
                onSaved = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
        }
    }
}

/** Tab bottom nav — 5 destinasi utama. Konflik & Add di luar bar (badge / FAB). */
private data class Tab(val route: String, val label: String, val icon: ImageVector)

private val TABS = listOf(
    Tab(Routes.HOME, "Akun", Icons.Outlined.AccountBalanceWallet),
    Tab(Routes.DASHBOARD, "Grafik", Icons.Outlined.Dashboard),
    Tab(Routes.HISTORY, "Riwayat", Icons.Outlined.ReceiptLong),
    Tab(Routes.CATEGORIES, "Kategori", Icons.Outlined.Category),
    Tab(Routes.INVESTMENTS, "Investasi", Icons.Outlined.TrendingUp),
)
private val TAB_ROUTES = TABS.map { it.route }.toSet()

@Composable
private fun KasKuBottomBar(nav: NavHostController, currentRoute: String?) {
    NavigationBar(containerColor = MaterialTheme.colorScheme.surface) {
        TABS.forEach { tab ->
            NavigationBarItem(
                selected = currentRoute == tab.route,
                onClick = {
                    if (currentRoute != tab.route) {
                        nav.navigate(tab.route) {
                            // Satu instance per tab; simpan/pulihkan state antar-tab.
                            popUpTo(nav.graph.findStartDestination().id) { saveState = true }
                            launchSingleTop = true
                            restoreState = true
                        }
                    }
                },
                icon = { Icon(tab.icon, contentDescription = tab.label) },
                label = { Text(tab.label) },
            )
        }
    }
}
