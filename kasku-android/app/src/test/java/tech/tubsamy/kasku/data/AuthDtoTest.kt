package tech.tubsamy.kasku.data

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.ErrorEnvelope
import tech.tubsamy.kasku.data.remote.dto.LoginData

class AuthDtoTest {

    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

    @Test
    fun parsesLoginSuccessEnvelope() {
        val raw = """{"success":true,"data":{"access_token":"abc.def","token_type":"Bearer","expires_in":900}}"""
        val env = json.decodeFromString<ApiEnvelope<LoginData>>(raw)
        assertTrue(env.success)
        assertEquals("abc.def", env.data?.access_token)
        assertEquals(900L, env.data?.expires_in)
    }

    @Test
    fun parsesErrorEnvelope() {
        val raw = """{"success":false,"error":{"code":"INVALID_CREDENTIALS","message":"Email atau password salah."}}"""
        val env = json.decodeFromString<ErrorEnvelope>(raw)
        assertEquals("Email atau password salah.", env.error?.message)
    }
}
