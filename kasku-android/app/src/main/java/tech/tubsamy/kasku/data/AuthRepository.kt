package tech.tubsamy.kasku.data

import kotlinx.serialization.json.Json
import retrofit2.HttpException
import tech.tubsamy.kasku.data.remote.ApiErrors
import tech.tubsamy.kasku.data.remote.AuthApi
import tech.tubsamy.kasku.data.remote.dto.LoginRequest
import java.io.IOException

class AuthRepository(
    private val api: AuthApi,
    private val tokenStore: TokenStore,
    private val json: Json,
) {

    /** Login → simpan access token. Result.failure membawa pesan siap-tampil. */
    suspend fun login(email: String, password: String): Result<Unit> {
        return try {
            val res = api.login(LoginRequest(email, password))
            val token = res.data?.access_token
            if (token.isNullOrBlank()) {
                Result.failure(Exception(res.error?.message ?: "Login gagal."))
            } else {
                tokenStore.saveAccessToken(token)
                Result.success(Unit)
            }
        } catch (e: HttpException) {
            Result.failure(Exception(ApiErrors.message(e, json, "Email atau password salah.")))
        } catch (e: IOException) {
            Result.failure(Exception("Tak bisa terhubung ke server. Periksa koneksi."))
        }
    }

    suspend fun logout() = tokenStore.clear()

    fun isLoggedIn(): Boolean = tokenStore.cachedAccessToken() != null
}
