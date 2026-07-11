package tech.tubsamy.kasku.data

import kotlinx.serialization.json.Json
import retrofit2.HttpException
import tech.tubsamy.kasku.data.remote.ApiErrors
import tech.tubsamy.kasku.data.remote.FinanceApi
import tech.tubsamy.kasku.data.remote.dto.AccountDto
import java.io.IOException

class AccountsRepository(
    private val api: FinanceApi,
    private val json: Json,
) {
    suspend fun listAccounts(): Result<List<AccountDto>> {
        return try {
            Result.success(api.listAccounts().data ?: emptyList())
        } catch (e: HttpException) {
            // 401 = access token 15m kadaluwarsa. ponytail: alur refresh belum ada (M1.5) →
            // untuk sekarang minta login ulang. Tambah OkHttp Authenticator saat refresh dibangun.
            val msg = if (e.code() == 401) {
                "Sesi berakhir. Silakan masuk lagi."
            } else {
                ApiErrors.message(e, json, "Gagal memuat akun.")
            }
            Result.failure(Exception(msg))
        } catch (e: IOException) {
            Result.failure(Exception("Tak bisa terhubung ke server."))
        }
    }
}
