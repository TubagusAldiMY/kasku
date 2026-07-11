package tech.tubsamy.kasku.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DashboardSummaryTest {

    private fun acc(balance: Long) =
        AccountItem(id = "a", name = "n", accountType = "CASH", balance = balance, currency = "IDR")

    private fun tx(type: String, amt: Long, cat: String = "Umum") =
        TransactionItem(id = "t$amt$type", type = type, amountIdr = amt, date = "2026-07-01", categoryId = null, categoryName = cat, notes = null)

    private val fmt: (Long) -> String = { "Rp $it" }

    @Test
    fun netWorth_and_savingsRate() {
        val s = DashboardSummary.compute(
            accounts = listOf(acc(1_000), acc(500)),
            txs = listOf(tx("INCOME", 200), tx("EXPENSE", 50)),
            formatMoney = fmt,
        )
        assertEquals(1_500, s.netWorth)
        assertEquals(200, s.monthIncome)
        assertEquals(50, s.monthExpense)
        assertEquals(75, s.savingsRate) // (200-50)/200 = 75%
    }

    @Test
    fun income_zero_guards_division() {
        val s = DashboardSummary.compute(emptyList(), listOf(tx("EXPENSE", 100)), fmt)
        assertEquals(0, s.savingsRate)
        assertTrue(s.insights[0].contains("melebihi"))
    }

    @Test
    fun overspend_clamps_negative_rate_to_zero() {
        val s = DashboardSummary.compute(emptyList(), listOf(tx("INCOME", 100), tx("EXPENSE", 300)), fmt)
        assertEquals(0, s.savingsRate) // (100-300)/100 = -200 → clamp 0
        assertTrue(s.insights[0].contains("melebihi"))
    }

    @Test
    fun biggest_expense_category_insight() {
        val s = DashboardSummary.compute(
            emptyList(),
            listOf(tx("EXPENSE", 100, "Makan"), tx("EXPENSE", 300, "Transport"), tx("INCOME", 500)),
            fmt,
        )
        assertTrue(s.insights.last().contains("Transport")) // 300 > 100 (Makan)
        assertTrue(s.insights.last().contains("Rp 300"))
    }
}
