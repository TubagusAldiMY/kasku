package tech.tubsamy.kasku.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DashboardSummaryTest {

    private fun acc(balance: Long) =
        AccountItem(id = "a", name = "n", accountType = "CASH", balance = balance, currency = "IDR")

    private fun tx(type: String, amt: Long, cat: String = "Umum", date: String = "2026-07-01") =
        TransactionItem(
            id = "t$amt$type$date",
            type = type,
            amountIdr = amt,
            date = date,
            categoryId = null,
            categoryName = cat,
            notes = null,
            accountId = null,
        )

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

    @Test
    fun trend_fills_empty_months_and_spans_year_boundary() {
        val t = DashboardCharts.trend(
            txs = listOf(
                tx("INCOME", 100, date = "2025-12-15"),
                tx("EXPENSE", 40, date = "2025-12-20"),
                tx("INCOME", 200, date = "2026-02-01"),
                tx("EXPENSE", 999, date = "2026-05-01"), // di luar window → diabaikan
            ),
            currentMonth = "2026-02",
            months = 3,
        )
        assertEquals(listOf("2025-12", "2026-01", "2026-02"), t.map { it.month })
        assertEquals(100, t[0].income); assertEquals(40, t[0].expense)
        assertEquals(0, t[1].income); assertEquals(0, t[1].expense) // Januari kosong
        assertEquals(200, t[2].income); assertEquals(0, t[2].expense)
    }

    @Test
    fun expenseByCategory_sums_sorts_desc_and_ignores_income() {
        val slices = DashboardCharts.expenseByCategory(
            listOf(
                tx("EXPENSE", 100, "Makan"),
                tx("EXPENSE", 50, "Makan"),
                tx("EXPENSE", 300, "Transport"),
                tx("INCOME", 999, "Gaji"), // bukan expense → diabaikan
            ),
        )
        assertEquals(listOf("Transport", "Makan"), slices.map { it.name })
        assertEquals(300, slices[0].total)
        assertEquals(150, slices[1].total) // 100 + 50
    }
}
