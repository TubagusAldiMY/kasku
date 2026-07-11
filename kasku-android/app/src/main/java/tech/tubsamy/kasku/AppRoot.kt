package tech.tubsamy.kasku

import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.ui.auth.LoginScreen
import tech.tubsamy.kasku.ui.auth.LoginViewModel
import tech.tubsamy.kasku.ui.auth.RegisterScreen
import tech.tubsamy.kasku.ui.auth.RegisterViewModel
import tech.tubsamy.kasku.ui.home.HomeScreen
import tech.tubsamy.kasku.ui.home.HomeViewModel
import tech.tubsamy.kasku.ui.transaction.AddTransactionScreen
import tech.tubsamy.kasku.ui.transaction.AddTransactionViewModel

private object Routes {
    const val LOGIN = "login"
    const val REGISTER = "register"
    const val HOME = "home"
    const val ADD_TRANSACTION = "add_transaction"
}

@Composable
fun AppRoot(container: AppContainer) {
    val nav = rememberNavController()
    val scope = rememberCoroutineScope()
    val appContext = LocalContext.current.applicationContext
    val start = if (container.authRepository.isLoggedIn()) Routes.HOME else Routes.LOGIN

    NavHost(navController = nav, startDestination = start) {
        composable(Routes.LOGIN) {
            val vm: LoginViewModel = viewModel(factory = LoginViewModel.factory(container.authRepository))
            LoginScreen(
                vm = vm,
                onLoggedIn = {
                    nav.navigate(Routes.HOME) {
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
                factory = HomeViewModel.factory(container.accountsRepository, appContext),
            )
            HomeScreen(
                vm = vm,
                onAddTransaction = { nav.navigate(Routes.ADD_TRANSACTION) },
                onLogout = {
                    scope.launch {
                        container.authRepository.logout()
                        nav.navigate(Routes.LOGIN) {
                            popUpTo(Routes.HOME) { inclusive = true }
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
                ),
            )
            AddTransactionScreen(
                vm = vm,
                onSaved = { nav.popBackStack() },
                onCancel = { nav.popBackStack() },
            )
        }
    }
}
