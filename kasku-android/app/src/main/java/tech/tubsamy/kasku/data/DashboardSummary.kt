package tech.tubsamy.kasku.data

/**
 * Ringkasan dashboard (F1) dihitung LOKAL dari Room — offline-first, tanpa endpoint backend.
 * Pure & KMP-ready (tanpa import Android). Unit-testable tanpa Room.
 *
 * insights = teks templated rule-based (bukan format Rupiah di sini agar tetap pure;
 * formatIdr diterapkan UI). insightExpense sudah diformat pemanggil via formatIdr injected.
 */
data class DashboardSummary(
    val netWorth: Long,      // Σ saldo akun aktif (owner F1: TANPA investasi/hutang)
    val monthIncome: Long,
    val monthExpense: Long,
    val savingsRate: Int,    // 0..100, clamp <0 → 0
    val insights: List<String>,
) {
    companion object {
        /**
         * @param formatMoney injeksi formatter (mis. ::formatIdr) supaya holder tetap pure/KMP.
         */
        fun compute(
            accounts: List<AccountItem>,
            txs: List<TransactionItem>,
            formatMoney: (Long) -> String,
        ): DashboardSummary {
            val netWorth = accounts.sumOf { it.balance }
            val income = txs.filter { it.type == "INCOME" }.sumOf { it.amountIdr }
            val expense = txs.filter { it.type == "EXPENSE" }.sumOf { it.amountIdr }

            // Guard divide-by-zero: income ≤ 0 → 0. Clamp negatif → 0.
            val savingsRate = if (income <= 0L) 0
            else (((income - expense) * 100 / income).toInt()).coerceAtLeast(0)

            val insights = buildList {
                add(
                    when {
                        savingsRate > 0 ->
                            "Kamu menyisihkan $savingsRate% dari pemasukan bulan ini. Pertahankan."
                        expense > income ->
                            "Pengeluaran melebihi pemasukan bulan ini."
                        else ->
                            "Pemasukan dan pengeluaran seimbang bulan ini."
                    },
                )
                // insight2 opsional: kategori pengeluaran terbesar.
                txs.filter { it.type == "EXPENSE" }
                    .groupBy { it.categoryName }
                    .mapValues { (_, list) -> list.sumOf { it.amountIdr } }
                    .maxByOrNull { it.value }
                    ?.let { (name, total) ->
                        add("Pengeluaran terbesar: $name (${formatMoney(total)}).")
                    }
            }

            return DashboardSummary(netWorth, income, expense, savingsRate, insights)
        }
    }
}
