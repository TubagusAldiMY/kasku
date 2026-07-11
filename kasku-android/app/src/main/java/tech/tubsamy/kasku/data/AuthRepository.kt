package tech.tubsamy.kasku.data

import kotlinx.serialization.json.Json
import retrofit2.HttpException
import tech.tubsamy.kasku.data.remote.ApiErrors
import tech.tubsamy.kasku.data.remote.AuthApi
import tech.tubsamy.kasku.data.remote.PersistentCookieJar
import tech.tubsamy.kasku.data.remote.dto.GoogleRequest
import tech.tubsamy.kasku.data.remote.dto.LoginRequest
import tech.tubsamy.kasku.data.remote.dto.RegisterRequest
import java.io.IOException

class AuthRepository(
    private val api: AuthApi,
    private val tokenStore: TokenStore,
    private val cookieJar: PersistentCookieJar,
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

    /** Register akun baru. Result.success membawa pesan (perlu verifikasi email — tak auto-login). */
    suspend fun register(email: String, username: String, password: String): Result<String> {
        return try {
            val res = api.register(RegisterRequest(email, username, password))
            if (res.success) {
                Result.success(res.data?.message ?: "Registrasi berhasil. Cek email untuk verifikasi.")
            } else {
                Result.failure(Exception(res.error?.message ?: "Registrasi gagal."))
            }
        } catch (e: HttpException) {
            Result.failure(Exception(ApiErrors.message(e, json, "Registrasi gagal.")))
        } catch (e: IOException) {
            Result.failure(Exception("Tak bisa terhubung ke server. Periksa koneksi."))
        }
    }

    /** Login/auto-register via Google ID token → simpan access token (seperti login biasa). */
    suspend fun googleLogin(idToken: String): Result<Unit> {
        return try {
            val token = api.google(GoogleRequest(idToken)).data?.access_token
            if (token.isNullOrBlank()) {
                Result.failure(Exception("Login Google gagal."))
            } else {
                tokenStore.saveAccessToken(token)
                Result.success(Unit)
            }
        } catch (e: HttpException) {
            Result.failure(Exception(ApiErrors.message(e, json, "Login Google gagal.")))
        } catch (e: IOException) {
            Result.failure(Exception("Tak bisa terhubung ke server. Periksa koneksi."))
        }
    }

    suspend fun logout() {
        tokenStore.clear()
        cookieJar.clear() // buang refresh token lokal juga
    }

    fun isLoggedIn(): Boolean = tokenStore.cachedAccessToken() != null
}
