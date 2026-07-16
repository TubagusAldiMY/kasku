package tech.tubsamy.kasku.data

import tech.tubsamy.kasku.data.remote.UserApi
import tech.tubsamy.kasku.data.remote.dto.ProfileDto

/** Profil user — read-only. load() throw bila gagal → VM tampilkan error. */
class ProfileRepository(
    private val api: UserApi,
) {
    suspend fun load(): ProfileDto = api.getProfile().data ?: ProfileDto()
}
