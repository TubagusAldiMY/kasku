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

/** Satu titik tren bulanan. month = "YYYY-MM". */
data class MonthlyPoint(val month: String, val income: Long, val expense: Long)

/** Satu irisan donat pengeluaran per kategori. */
data class CategorySlice(val name: String, val total: Long)

/**
 * Data grafik dashboard — dihitung LOKAL dari Room (offline-first), pure & KMP-ready.
 * trend & breakdown dipisah dari DashboardSummary karena butuh rentang data berbeda
 * (trend = beberapa bulan; breakdown = bulan berjalan).
 */
data class DashboardCharts(
    val trend: List<MonthlyPoint>,
    val expenseByCategory: List<CategorySlice>,
) {
    companion object {
        val EMPTY = DashboardCharts(emptyList(), emptyList())

        /**
         * Tren income vs expense untuk [months] bulan terakhir sampai (inklusif) [currentMonth]
         * ("YYYY-MM"). Bulan tanpa transaksi tetap muncul dengan 0/0 supaya sumbu-x kontinu.
         *
         * date TransactionItem = "YYYY-MM-DD"; kunci bulan diambil dari 7 char pertama.
         */
        fun trend(txs: List<TransactionItem>, currentMonth: String, months: Int): List<MonthlyPoint> {
            require(months >= 1) { "months must be >= 1" }
            val keys = lastMonthKeys(currentMonth, months)
            val window = keys.toSet()
            val grouped = txs
                .filter { it.date.take(7) in window }
                .groupBy { it.date.take(7) }
            return keys.map { m ->
                val list = grouped[m].orEmpty()
                MonthlyPoint(
                    month = m,
                    income = list.filter { it.type == "INCOME" }.sumOf { it.amountIdr },
                    expense = list.filter { it.type == "EXPENSE" }.sumOf { it.amountIdr },
                )
            }
        }

        /** Breakdown pengeluaran per kategori (desc). txs sudah difilter ke bulan berjalan oleh pemanggil. */
        fun expenseByCategory(monthTxs: List<TransactionItem>): List<CategorySlice> =
            monthTxs.filter { it.type == "EXPENSE" }
                .groupBy { it.categoryName }
                .map { (name, list) -> CategorySlice(name, list.sumOf { it.amountIdr }) }
                .filter { it.total > 0 }
                .sortedByDescending { it.total }

        /** ["YYYY-MM", ...] naik kronologis, panjang [count], berakhir di [currentMonth]. */
        private fun lastMonthKeys(currentMonth: String, count: Int): List<String> {
            val year = currentMonth.substring(0, 4).toInt()
            val month = currentMonth.substring(5, 7).toInt() // 1..12
            var idx = year * 12 + (month - 1) // bulan absolut sejak tahun 0
            val out = ArrayDeque<String>()
            repeat(count) {
                val y = idx / 12
                val mo = idx % 12 + 1
                out.addFirst("%04d-%02d".format(y, mo))
                idx--
            }
            return out.toList()
        }
    }
}
