package tech.tubsamy.kasku.data.remote

import android.content.Context
import androidx.credentials.CredentialManager
import androidx.credentials.GetCredentialRequest
import com.google.android.libraries.identity.googleid.GetGoogleIdOption
import com.google.android.libraries.identity.googleid.GoogleIdTokenCredential
import tech.tubsamy.kasku.BuildConfig

/**
 * Ambil Google ID token native lewat Credential Manager (pengganti flow redirect web).
 * ID token dikirim ke backend POST /auth/google.
 *
 * Prasyarat Google Console: OAuth client tipe Android (package tech.tubsamy.kasku + SHA-1)
 * harus terdaftar; serverClientId = Web client ID (GOOGLE_SERVER_CLIENT_ID).
 */
object GoogleAuthClient {

    suspend fun getIdToken(context: Context): String {
        val googleIdOption = GetGoogleIdOption.Builder()
            .setServerClientId(BuildConfig.GOOGLE_SERVER_CLIENT_ID)
            .setFilterByAuthorizedAccounts(false) // tampilkan semua akun, bukan hanya yang pernah login
            .setAutoSelectEnabled(false)
            .build()

        val request = GetCredentialRequest.Builder()
            .addCredentialOption(googleIdOption)
            .build()

        val result = CredentialManager.create(context).getCredential(context, request)
        val credential = result.credential

        if (credential.type != GoogleIdTokenCredential.TYPE_GOOGLE_ID_TOKEN_CREDENTIAL) {
            error("Kredensial bukan Google ID token")
        }
        return GoogleIdTokenCredential.createFrom(credential.data).idToken
    }
}
