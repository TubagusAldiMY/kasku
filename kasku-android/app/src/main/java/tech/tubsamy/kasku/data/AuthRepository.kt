package tech.tubsamy.kasku.data

import kotlinx.serialization.json.Json
import retrofit2.HttpException
import tech.tubsamy.kasku.data.remote.ApiErrors
import tech.tubsamy.kasku.data.remote.AuthApi
import tech.tubsamy.kasku.data.remote.PersistentCookieJar
import tech.tubsamy.kasku.data.remote.dto.ChangePasswordRequest
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

    /** Minta email reset password. Backend selalu balas generik (anti-enumeration). */
    suspend fun forgotPassword(email: String): Result<String> {
        return try {
            val res = api.forgotPassword(tech.tubsamy.kasku.data.remote.dto.ForgotPasswordRequest(email))
            Result.success(res.data?.message ?: "Jika email terdaftar, instruksi reset password telah dikirim.")
        } catch (e: HttpException) {
            Result.failure(Exception(ApiErrors.message(e, json, "Gagal mengirim permintaan reset.")))
        } catch (e: IOException) {
            Result.failure(Exception("Tak bisa terhubung ke server. Periksa koneksi."))
        }
    }

    /** Reset password dengan token dari email. */
    suspend fun resetPassword(token: String, newPassword: String): Result<String> {
        return try {
            val res = api.resetPassword(
                tech.tubsamy.kasku.data.remote.dto.ResetPasswordRequest(token, newPassword),
            )
            if (res.success) {
                Result.success(res.data?.message ?: "Password berhasil direset. Silakan login.")
            } else {
                Result.failure(Exception(res.error?.message ?: "Reset password gagal."))
            }
        } catch (e: HttpException) {
            Result.failure(Exception(ApiErrors.message(e, json, "Token tidak valid atau kedaluwarsa.")))
        } catch (e: IOException) {
            Result.failure(Exception("Tak bisa terhubung ke server. Periksa koneksi."))
        }
    }

    /** Verifikasi email dengan token dari link/email. */
    suspend fun verifyEmail(token: String): Result<String> {
        return try {
            val res = api.verifyEmail(token)
            if (res.success) {
                Result.success(res.data?.message ?: "Email berhasil diverifikasi. Silakan login.")
            } else {
                Result.failure(Exception(res.error?.message ?: "Verifikasi email gagal."))
            }
        } catch (e: HttpException) {
            Result.failure(Exception(ApiErrors.message(e, json, "Token verifikasi tidak valid atau kedaluwarsa.")))
        } catch (e: IOException) {
            Result.failure(Exception("Tak bisa terhubung ke server. Periksa koneksi."))
        }
    }

    /** Kirim ulang email verifikasi. Backend selalu balas generik (anti-enumeration). */
    suspend fun resendVerification(email: String): Result<String> {
        return try {
            val res = api.resendVerification(tech.tubsamy.kasku.data.remote.dto.ResendVerificationRequest(email))
            Result.success(res.data?.message ?: "Jika email terdaftar, link verifikasi baru telah dikirim.")
        } catch (e: HttpException) {
            Result.failure(Exception(ApiErrors.message(e, json, "Gagal mengirim ulang verifikasi.")))
        } catch (e: IOException) {
            Result.failure(Exception("Tak bisa terhubung ke server. Periksa koneksi."))
        }
    }

    /** Ubah password (butuh sesi login aktif). Pesan error jujur dari server. */
    suspend fun changePassword(currentPassword: String, newPassword: String): Result<Unit> {
        return try {
            api.changePassword(ChangePasswordRequest(currentPassword, newPassword))
            Result.success(Unit)
        } catch (e: HttpException) {
            Result.failure(Exception(ApiErrors.message(e, json, "Gagal mengubah password.")))
        } catch (e: IOException) {
            Result.failure(Exception("Tak bisa terhubung ke server. Periksa koneksi."))
        }
    }
}
