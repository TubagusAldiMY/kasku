package tech.tubsamy.kasku.data.sync

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/** Port skenario conflict.spec — LWW decision tree. */
class ConflictResolverTest {

    private fun view(ts: String, deleted: Boolean = false, dirty: Boolean = false) =
        SyncableView(ts, deleted, dirty)

    @Test fun `local null applies server`() {
        val r = ConflictResolver.applyServerWins(null, view("2025-07-11T10:00:00Z"))
        assertEquals(Decision.APPLY_SERVER, r.decision)
        assertFalse(r.loserHadLocalChanges)
    }

    @Test fun `server tombstone always applies and flags dirty loser`() {
        val r = ConflictResolver.applyServerWins(
            local = view("2025-07-11T12:00:00Z", dirty = true),
            server = view("2025-07-11T09:00:00Z", deleted = true),
        )
        assertEquals(Decision.APPLY_SERVER, r.decision)
        assertTrue(r.loserHadLocalChanges)
    }

    @Test fun `server newer or equal applies`() {
        val equal = ConflictResolver.applyServerWins(
            view("2025-07-11T10:00:00Z"), view("2025-07-11T10:00:00Z"),
        )
        assertEquals(Decision.APPLY_SERVER, equal.decision)

        val newer = ConflictResolver.applyServerWins(
            view("2025-07-11T10:00:00Z", dirty = true), view("2025-07-11T11:00:00Z"),
        )
        assertEquals(Decision.APPLY_SERVER, newer.decision)
        assertTrue(newer.loserHadLocalChanges)
    }

    @Test fun `local newer keeps local`() {
        val r = ConflictResolver.applyServerWins(
            view("2025-07-11T12:00:00Z", dirty = true), view("2025-07-11T10:00:00Z"),
        )
        assertEquals(Decision.KEEP_LOCAL, r.decision)
        assertFalse(r.loserHadLocalChanges)
    }

    @Test fun `invalid server timestamp keeps local`() {
        val r = ConflictResolver.applyServerWins(view("2025-07-11T10:00:00Z"), view("not-a-date"))
        assertEquals(Decision.KEEP_LOCAL, r.decision)
    }

    @Test fun `invalid local timestamp applies server`() {
        val r = ConflictResolver.applyServerWins(view("garbage", dirty = true), view("2025-07-11T10:00:00Z"))
        assertEquals(Decision.APPLY_SERVER, r.decision)
        assertTrue(r.loserHadLocalChanges)
    }
}
