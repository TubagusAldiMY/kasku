package tech.tubsamy.kasku.data.sync

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull
import tech.tubsamy.kasku.data.ConflictItem
import tech.tubsamy.kasku.data.ConflictsRepository
import tech.tubsamy.kasku.data.local.KasKuDatabase

/**
 * Aksi "Pakai versi saya" (F4): re-enqueue nilai lokal dengan timestamp baru → menang LWW →
 * hapus konflik. Routing per-resource ke mutations.update yang sesuai (port pola web
 * enqueueUpdate(resource, entity_id, local_value)).
 *
 * "Terima server" (acceptServer) tetap di ConflictsRepository — tak butuh mutations.
 */
class ConflictResolutionService(
    @Suppress("unused") private val db: KasKuDatabase, // reserved: routing masa depan (investments)
    private val conflicts: ConflictsRepository,
    private val accountMutations: AccountMutations,
    private val transactionMutations: TransactionMutations,
    private val json: Json,
) {
    /**
     * Parse localValue JSON → panggil mutations.update resource terkait (regenerate sync_id +
     * updated_at=now → menang LWW) → hapus konflik.
     *
     * Edge case (golden web): entity lokal sudah hilang → mutations.update no-op → tetap hapus konflik.
     * investments: belum ada InvestmentMutations → keepLocal = no-op + hapus konflik.
     */
    suspend fun keepLocal(conflict: ConflictItem) {
        val local = runCatching { json.parseToJsonElement(conflict.localValue) as? JsonObject }
            .getOrNull()

        when (Resource.entries.firstOrNull { it.local == conflict.resource }) {
            Resource.ACCOUNTS -> local?.let { o ->
                accountMutations.update(
                    id = conflict.entityId,
                    name = o.strOrNull("name"),
                    accountType = o.strOrNull("account_type"),
                    balance = o.longFor("balance"),
                    currency = o.strOrNull("currency"),
                )
            }

            Resource.TRANSACTIONS -> local?.let { o ->
                transactionMutations.update(
                    id = conflict.entityId,
                    accountId = o.strOrNull("account_id"),
                    type = o.strOrNull("transaction_type"),
                    amountIdr = o.longFor("amount_idr"),
                    categoryId = o.strOrNull("category_id"),
                    date = o.strOrNull("transaction_date"),
                    notes = o.strOrNull("notes"),
                )
            }

            // ponytail: investment mutations menyusul → keepLocal = no-op, cukup hapus konflik.
            Resource.INVESTMENTS, null -> Unit
        }

        conflicts.acceptServer(conflict.id)
    }

    private fun JsonObject.strOrNull(key: String): String? =
        this[key]?.jsonPrimitive?.content?.takeIf { it != "null" }

    private fun JsonObject.longFor(key: String): Long? =
        this[key]?.jsonPrimitive?.longOrNull
}
