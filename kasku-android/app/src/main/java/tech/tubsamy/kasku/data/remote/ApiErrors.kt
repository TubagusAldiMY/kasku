package tech.tubsamy.kasku.data.remote

import kotlinx.serialization.json.Json
import retrofit2.HttpException
import tech.tubsamy.kasku.data.remote.dto.ErrorEnvelope

/** Ambil pesan error jujur dari body respons non-2xx (bukan "HTTP 401" mentah). */
object ApiErrors {
    fun message(e: HttpException, json: Json, fallback: String): String {
        val body = e.response()?.errorBody()?.string().orEmpty()
        return runCatching { json.decodeFromString<ErrorEnvelope>(body).error?.message }
            .getOrNull()
            ?: fallback
    }
}
