package tech.tubsamy.kasku.data.sync

/**
 * Port dari conflict.ts — Last-Writer-Wins berbasis timestamp updated_at (ISO 8601).
 * Pure logic, ZERO import Android/Room → KMP-ready & unit-testable tanpa Context.
 */

/** Pandangan minimal sebuah entity yang dibutuhkan LWW. */
data class SyncableView(
    val updatedAt: String,
    val deleted: Boolean,
    val localDirty: Boolean,
)

enum class Decision { APPLY_SERVER, KEEP_LOCAL }

data class ConflictResolution(
    val decision: Decision,
    val loserHadLocalChanges: Boolean,
)

object ConflictResolver {

    /**
     * Decision tree (persis golden reference):
     * 1. local null            → apply_server
     * 2. server tombstone      → apply_server (loser dirty jika local dirty)
     * 3. timestamp NaN         → server NaN keep_local; local NaN apply_server
     * 4. serverT >= localT     → apply_server (loser dirty jika local dirty)
     *    localT  >  serverT    → keep_local
     */
    fun applyServerWins(local: SyncableView?, server: SyncableView): ConflictResolution {
        if (local == null) {
            return ConflictResolution(Decision.APPLY_SERVER, loserHadLocalChanges = false)
        }
        if (server.deleted) {
            return ConflictResolution(Decision.APPLY_SERVER, loserHadLocalChanges = local.localDirty)
        }

        val localT = parseMillis(local.updatedAt)
        val serverT = parseMillis(server.updatedAt)

        if (serverT == null) {
            // Server timestamp invalid → jangan timpa local.
            return ConflictResolution(Decision.KEEP_LOCAL, loserHadLocalChanges = false)
        }
        if (localT == null) {
            return ConflictResolution(Decision.APPLY_SERVER, loserHadLocalChanges = local.localDirty)
        }

        return if (serverT >= localT) {
            ConflictResolution(Decision.APPLY_SERVER, loserHadLocalChanges = local.localDirty)
        } else {
            ConflictResolution(Decision.KEEP_LOCAL, loserHadLocalChanges = false)
        }
    }

    /** ISO 8601 → epoch millis, atau null jika tak terparse (setara Date.parse → NaN). */
    fun parseMillis(iso: String): Long? =
        runCatching { java.time.Instant.parse(iso).toEpochMilli() }.getOrNull()
}
