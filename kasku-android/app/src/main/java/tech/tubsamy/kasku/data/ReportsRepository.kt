package tech.tubsamy.kasku.data

import tech.tubsamy.kasku.data.remote.ReportApi
import tech.tubsamy.kasku.data.remote.dto.ReportDto

/**
 * Laporan keuangan — read-only, tak di-cache lokal (bukan bagian offline-sync).
 * load() throw bila gagal → VM tampilkan error + tombol coba lagi.
 */
class ReportsRepository(
    private val api: ReportApi,
) {
    suspend fun load(months: Int = 6): ReportDto = api.getReport(months = months).data ?: ReportDto()
}
