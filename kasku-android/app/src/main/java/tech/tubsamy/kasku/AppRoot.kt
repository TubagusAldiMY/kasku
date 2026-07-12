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
import tech.tubsamy.kasku.ui.auth.LoginScreen
import tech.tubsamy.kasku.ui.auth.LoginViewModel
import tech.tubsamy.kasku.ui.auth.RegisterScreen
import tech.tubsamy.kasku.ui.auth.RegisterViewModel
import tech.tubsamy.kasku.ui.category.CategoryScreen
import tech.tubsamy.kasku.ui.category.CategoryViewModel
import tech.tubsamy.kasku.ui.conflicts.ConflictsScreen
import tech.tubsamy.kasku.ui.conflicts.ConflictsViewModel
import tech.tubsamy.kasku.ui.dashboard.DashboardScreen
import tech.tubsamy.kasku.ui.dashboard.DashboardViewModel
import tech.tubsamy.kasku.ui.home.HomeScreen
import tech.tubsamy.kasku.ui.home.HomeViewModel
import tech.tubsamy.kasku.ui.investment.AddInvestmentScreen
import tech.tubsamy.kasku.ui.investment.AddInvestmentViewModel
import tech.tubsamy.kasku.ui.investment.InvestmentScreen
import tech.tubsamy.kasku.ui.investment.InvestmentViewModel
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
            )
        }
        composable(Routes.REGISTER) {
            val vm: RegisterViewModel = viewModel(factory = RegisterViewModel.factory(container.authRepository))
            RegisterScreen(
                vm = vm,
                onBackToLogin = { nav.popBackStack(Routes.LOGIN, inclusive = false) },
            )
        }
        composable(Routes.HOME) {
            val vm: HomeViewModel = viewModel(
                factory = HomeViewModel.factory(
                    container.accountsRepository,
                    container.conflictsRepository,
                    appContext,
                ),
            )
            HomeScreen(
                vm = vm,
                onConflicts = { nav.navigate(Routes.CONFLICTS) },
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
                factory = InvestmentViewModel.factory(container.investmentsRepository, appContext),
            )
            InvestmentScreen(
                vm = vm,
                onBack = { nav.popBackStack() },
                onAddInvestment = { nav.navigate(Routes.ADD_INVESTMENT) },
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
