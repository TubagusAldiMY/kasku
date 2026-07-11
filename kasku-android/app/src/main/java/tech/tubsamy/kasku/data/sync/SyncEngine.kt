package tech.tubsamy.kasku.data.sync

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import tech.tubsamy.kasku.data.remote.SyncApi
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.PushRequest
import tech.tubsamy.kasku.data.remote.dto.PushResponse
import tech.tubsamy.kasku.data.remote.dto.SyncOperation

/** Snapshot status sync untuk UI (badge queue/konflik/error). */
data class SyncStatus(
    val queuedCount: Int = 0,
    val conflictCount: Int = 0,
    val lastSyncAt: String? = null,
    val error: String? = null,
)

/**
 * Port dari engine.ts — 3 fase: PULL_BASELINE → PUSH_PENDING → PULL_POSTPUSH.
 * Generik 3-resource (accounts/transactions/investments).
 *
 * Bebas import Android/Room: bergantung pada [SyncStore] (port) + [SyncApi] (Retrofit
 * interface) + Json. Bisa diuji dengan fake store & fake api.
 */
class SyncEngine(
    private val store: SyncStore,
    private val api: SyncApi,
    private val json: Json,
    private val clock: () -> String = { java.time.Instant.now().toString() },
) {
    companion object {
        const val PUSH_BATCH_SIZE = 50
        const val PULL_DEFAULT_SINCE = "1970-01-01T00:00:00.000Z"
    }

    /** Siklus penuh. Melempar jika HTTP gagal (caller/worker menangani retry). */
    suspend fun syncAll(): SyncStatus {
        // FASE 1: pull baseline
        for (resource in SYNCABLE_RESOURCES) pullDelta(resource)

        // FASE 2: push pending
        pushPending()

        // FASE 3: pull post-push (server bisa set field tambahan saat push)
        for (resource in SYNCABLE_RESOURCES) pullDelta(resource)

        return status()
    }

    /** Reset baris IN_FLIGHT yatim ke PENDING. Panggil saat init. */
    suspend fun recoverStuck() = store.recoverStuck()

    suspend fun status(): SyncStatus =
        SyncStatus(
            queuedCount = store.queuedCount(),
            conflictCount = store.conflictCount(),
            lastSyncAt = clock(),
            error = null,
        )

    // --- FASE 1 & 3 ------------------------------------------------------

    private suspend fun pullDelta(resource: Resource) {
        val since = store.lastSyncedAt(resource) ?: PULL_DEFAULT_SINCE
        val body = unwrap(api.pull(since), "pull ${resource.local}")

        for (change in body.changes) {
            if (change.entity_type != resource.server) continue

            val incoming = SyncMapping.serverView(change.data)
            val local = store.localView(resource, change.entity_id)
            val res = ConflictResolver.applyServerWins(local, incoming)

            if (res.decision != Decision.APPLY_SERVER) continue

            if (res.loserHadLocalChanges) {
                store.recordConflict(
                    resource = resource,
                    entityId = change.entity_id,
                    origin = "pull",
                    localValue = localViewJson(local),
                    serverValue = change.data.toString(),
                )
            }

            if (incoming.deleted) {
                store.hardDelete(resource, change.entity_id)
            } else {
                store.upsertFromServer(resource, change.data)
            }
        }

        store.setLastSyncedAt(resource, body.server_timestamp)
    }

    // --- FASE 2 ----------------------------------------------------------

    private suspend fun pushPending() {
        val pending = store.pendingOps(PUSH_BATCH_SIZE)
        if (pending.isEmpty()) return

        val ops = pending.map { it.toOperation() }
        val localBySyncId = pending.associateBy({ it.syncId }, { it.payloadJson })
        val syncIds = pending.map { it.syncId }

        store.markInFlight(syncIds)

        val body: PushResponse = try {
            unwrap(api.push(PushRequest(ops)), "push")
        } catch (e: Exception) {
            // Network/HTTP gagal: seluruh batch FAILED → attempts++, retry di sync berikut.
            val msg = e.message ?: "push gagal"
            for (id in syncIds) store.markFailed(id, msg)
            throw e
        }

        for (result in body.results) {
            when (result.status) {
                "applied", "skipped" -> store.markAcked(result.sync_id)

                "conflict" -> {
                    val serverData = result.server_data
                    if (serverData != null) {
                        val resource = Resource.fromServer(result.entity_type)
                        if (resource != null) {
                            store.recordConflict(
                                resource = resource,
                                entityId = result.entity_id,
                                origin = "push",
                                localValue = localBySyncId[result.sync_id] ?: "null",
                                serverValue = serverData.toString(),
                            )
                            val view = SyncMapping.serverView(serverData)
                            if (view.deleted) {
                                store.hardDelete(resource, result.entity_id)
                            } else {
                                store.upsertFromServer(resource, serverData)
                            }
                        }
                    }
                    store.markAcked(result.sync_id)
                }

                else -> store.markFailed(result.sync_id, "server reported error")
            }
        }
    }

    // --- helpers ---------------------------------------------------------

    private fun QueuedOp.toOperation(): SyncOperation {
        val payload = json.parseToJsonElement(payloadJson)
        return SyncOperation(
            sync_id = syncId,
            entity_type = resource.server,
            entity_id = entityId,
            operation = operation.lowercase(),
            payload = payload,
            client_timestamp = clock(),
        )
    }

    /** JSON string dari view lokal yang kalah (untuk kolom local_value di sync_conflicts). */
    private fun localViewJson(local: SyncableView?): String {
        if (local == null) return "null"
        return buildJsonObject {
            put("updated_at", JsonPrimitive(local.updatedAt))
            put("deleted", JsonPrimitive(local.deleted))
            put("local_dirty", JsonPrimitive(local.localDirty))
        }.toString()
    }

    /** Envelope { success, data, error } → data, atau lempar dengan pesan error. */
    private fun <T> unwrap(env: ApiEnvelope<T>, op: String): T {
        if (!env.success || env.data == null) {
            val msg = env.error?.message ?: "sync $op gagal (envelope tanpa data)"
            throw IllegalStateException(msg)
        }
        return env.data
    }
}
