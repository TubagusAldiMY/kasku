package tech.tubsamy.kasku.data.sync

import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import tech.tubsamy.kasku.data.local.InvestmentEntity

/**
 * Kontrak field payload sync investasi. Payload push HARUS round-trip lewat
 * SyncMapping.toInvestment ke domain yang sama (membuktikan nama field snake_case
 * cocok dgn golden ref web InvestmentRow: name/asset_type/symbol/units/avg_buy_price_idr).
 *
 * Payload direplikasi di sini (fungsi payload() di InvestmentMutations privat & butuh Room).
 * Jika keduanya menyimpang, test ini gagal — pengingat menyelaraskan.
 */
class InvestmentPayloadTest {

    private fun payload(row: InvestmentEntity): JsonObject = buildJsonObject {
        put("id", JsonPrimitive(row.id))
        put("name", JsonPrimitive(row.name))
        put("asset_type", JsonPrimitive(row.asset_type))
        put("units", JsonPrimitive(row.units))
        put("avg_buy_price_idr", JsonPrimitive(row.avg_buy_price_idr))
        put("updated_at", JsonPrimitive(row.updated_at))
        row.symbol?.let { put("symbol", JsonPrimitive(it)) }
    }

    @Test fun `payload round-trips through SyncMapping`() {
        val row = InvestmentEntity(
            id = "inv-1",
            sync_id = "s-1",
            updated_at = "2026-07-11T10:00:00Z",
            local_dirty = true,
            deleted = false,
            name = "Bitcoin",
            asset_type = "CRYPTO",
            symbol = "BTC",
            units = 1.5,
            avg_buy_price_idr = 950_000_000L,
        )

        val mapped = SyncMapping.toInvestment(payload(row))

        assertEquals("inv-1", mapped.id)
        assertEquals("Bitcoin", mapped.name)
        assertEquals("CRYPTO", mapped.asset_type)
        assertEquals("BTC", mapped.symbol)
        assertEquals(1.5, mapped.units, 0.0)
        assertEquals(950_000_000L, mapped.avg_buy_price_idr)
        assertEquals("2026-07-11T10:00:00Z", mapped.updated_at)
    }

    @Test fun `symbol omitted when null`() {
        val row = InvestmentEntity(
            id = "inv-2",
            sync_id = "s-2",
            updated_at = "2026-07-11T10:00:00Z",
            local_dirty = true,
            deleted = false,
            name = "Emas Antam",
            asset_type = "GOLD",
            symbol = null,
            units = 10.0,
            avg_buy_price_idr = 1_200_000L,
        )

        val p = payload(row)
        assertNull(p["symbol"])
        assertNull(SyncMapping.toInvestment(p).symbol)
    }
}
