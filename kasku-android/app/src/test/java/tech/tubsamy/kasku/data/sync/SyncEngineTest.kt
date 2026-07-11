package tech.tubsamy.kasku.data.sync

import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import tech.tubsamy.kasku.data.remote.SyncApi
import tech.tubsamy.kasku.data.remote.dto.ApiEnvelope
import tech.tubsamy.kasku.data.remote.dto.EntityChange
import tech.tubsamy.kasku.data.remote.dto.PullResponse
import tech.tubsamy.kasku.data.remote.dto.PushRequest
import tech.tubsamy.kasku.data.remote.dto.PushResponse
import tech.tubsamy.kasku.data.remote.dto.SyncResult

/** Fake in-memory store — tanpa Room/Context. */
private class FakeSyncStore : SyncStore {
    val meta = mutableMapOf<String, String>()
    val locals = mutableMapOf<Pair<String, String>, SyncableView>() // (resource, id) -> view
    val upserts = mutableListOf<Pair<Resource, JsonElement>>()
    val deletes = mutableListOf<Pair<Resource, String>>()
    val conflicts = mutableListOf<Triple<Resource, String, String>>() // resource, id, origin
    var queue = mutableListOf<QueuedOp>()
    val acked = mutableListOf<String>()
    val failed = mutableListOf<Pair<String, String>>()
    val inFlight = mutableListOf<String>()
    var recovered = false

    override suspend fun lastSyncedAt(resource: Resource) = meta[resource.local]
    override suspend fun setLastSyncedAt(resource: Resource, iso: String) { meta[resource.local] = iso }
    override suspend fun localView(resource: Resource, id: String) = locals[resource.local to id]
    override suspend fun upsertFromServer(resource: Resource, data: JsonElement) { upserts += resource to data }
    override suspend fun hardDelete(resource: Resource, id: String) { deletes += resource to id }
    override suspend fun pendingOps(limit: Int) = queue.take(limit)
    override suspend fun markInFlight(syncIds: List<String>) { inFlight += syncIds }
    override suspend fun markAcked(syncId: String) { acked += syncId; queue.removeAll { it.syncId == syncId } }
    override suspend fun markFailed(syncId: String, error: String) { failed += syncId to error }
    override suspend fun recoverStuck() { recovered = true }
    override suspend fun queuedCount() = queue.size
    override suspend fun recordConflict(
        resource: Resource, entityId: String, origin: String, localValue: String, serverValue: String,
    ) { conflicts += Triple(resource, entityId, origin) }
    override suspend fun conflictCount() = conflicts.size
}

/** Fake SyncApi — scripted pull/push responses. */
private class FakeSyncApi(
    val pullChanges: List<EntityChange> = emptyList(),
    val pushResults: List<SyncResult> = emptyList(),
) : SyncApi {
    var pushCalls = 0
    // Hormati cursor `since` seperti server asli: hanya kirim change yang lebih baru dari cursor.
    // Tanpa ini, PULL_POSTPUSH (fase 3) mengembalikan change yang sama → upsert terhitung dobel.
    override suspend fun pull(since: String): ApiEnvelope<PullResponse> =
        ApiEnvelope(
            success = true,
            data = PullResponse(pullChanges.filter { it.updated_at > since }, "2025-07-11T10:00:00Z"),
        )
    override suspend fun push(body: PushRequest): ApiEnvelope<PushResponse> {
        pushCalls++
        return ApiEnvelope(
            success = true,
            data = PushResponse(
                processed = pushResults.count { it.status == "applied" },
                conflicts = pushResults.count { it.status == "conflict" },
                skipped = pushResults.count { it.status == "skipped" },
                results = pushResults,
                server_timestamp = "2025-07-11T10:00:01Z",
            ),
        )
    }
}

class SyncEngineTest {
    private val json = Json { ignoreUnknownKeys = true }
    private fun jel(s: String): JsonElement = json.parseToJsonElement(s)

    @Test fun `pull upserts account and advances sync_meta`() = runTest {
        val store = FakeSyncStore()
        val api = FakeSyncApi(
            pullChanges = listOf(
                EntityChange(
                    entity_type = "financial_account",
                    entity_id = "a1",
                    operation = "update",
                    data = jel("""{"id":"a1","name":"Kas","account_type":"cash","balance":1000,"currency":"IDR","updated_at":"2025-07-11T09:00:00Z"}"""),
                    updated_at = "2025-07-11T09:00:00Z",
                ),
            ),
        )
        SyncEngine(store, api, json).syncAll()

        assertEquals(1, store.upserts.count { it.first == Resource.ACCOUNTS })
        assertEquals("2025-07-11T10:00:00Z", store.meta["accounts"])
    }

    @Test fun `pull with dirty local loser records conflict`() = runTest {
        val store = FakeSyncStore().apply {
            locals["accounts" to "a1"] = SyncableView("2025-07-11T08:00:00Z", deleted = false, localDirty = true)
        }
        val api = FakeSyncApi(
            pullChanges = listOf(
                EntityChange(
                    "financial_account", "a1", "update",
                    jel("""{"id":"a1","name":"Kas","account_type":"cash","balance":1,"currency":"IDR","updated_at":"2025-07-11T09:00:00Z"}"""),
                    "2025-07-11T09:00:00Z",
                ),
            ),
        )
        SyncEngine(store, api, json).syncAll()
        assertTrue(store.conflicts.any { it.third == "pull" })
    }

    @Test fun `pull tombstone hard-deletes`() = runTest {
        val store = FakeSyncStore()
        val api = FakeSyncApi(
            pullChanges = listOf(
                EntityChange(
                    "financial_account", "a1", "delete",
                    jel("""{"id":"a1","_deleted":true,"updated_at":"2025-07-11T09:00:00Z"}"""),
                    "2025-07-11T09:00:00Z",
                ),
            ),
        )
        SyncEngine(store, api, json).syncAll()
        assertTrue(store.deletes.any { it.second == "a1" })
    }

    @Test fun `push applied acks and dequeues (idempotent single push)`() = runTest {
        val store = FakeSyncStore().apply {
            queue = mutableListOf(
                QueuedOp("s1", Resource.ACCOUNTS, "CREATE", "a1",
                    """{"id":"a1","name":"Kas","account_type":"cash","balance":1000,"currency":"IDR","updated_at":"2025-07-11T09:00:00Z"}"""),
            )
        }
        val api = FakeSyncApi(
            pushResults = listOf(SyncResult("s1", "financial_account", "a1", "applied")),
        )
        SyncEngine(store, api, json).syncAll()

        assertTrue(store.acked.contains("s1"))
        assertEquals(1, api.pushCalls)              // dedup: satu batch push
        assertEquals(0, store.queue.size)
    }

    @Test fun `push conflict records conflict and applies server_data`() = runTest {
        val store = FakeSyncStore().apply {
            queue = mutableListOf(
                QueuedOp("s1", Resource.ACCOUNTS, "UPDATE", "a1", """{"id":"a1","balance":5}"""),
            )
        }
        val api = FakeSyncApi(
            pushResults = listOf(
                SyncResult(
                    "s1", "financial_account", "a1", "conflict",
                    server_data = jel("""{"id":"a1","name":"Srv","account_type":"cash","balance":9,"currency":"IDR","updated_at":"2025-07-11T10:00:00Z"}"""),
                ),
            ),
        )
        SyncEngine(store, api, json).syncAll()

        assertTrue(store.conflicts.any { it.third == "push" })
        assertTrue(store.upserts.any { it.first == Resource.ACCOUNTS })
        assertTrue(store.acked.contains("s1"))
    }
}
