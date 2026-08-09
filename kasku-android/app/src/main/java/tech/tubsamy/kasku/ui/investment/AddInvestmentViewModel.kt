package tech.tubsamy.kasku.ui.investment

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.InvestmentsRepository

/**
 * Form tambah/edit instrumen (ONLINE). editId != null → mode edit: hanya name/asset_type/symbol
 * yang bisa diubah (units & avg dikelola lewat pencatatan BUY/SELL). Prefill dari cache repo
 * (refresh dulu supaya terisi bila layar dibuka langsung).
 */
class AddInvestmentViewModel(
    private val repo: InvestmentsRepository,
    private val editId: String? = null,
) : ViewModel() {

    val isEdit: Boolean get() = editId != null

    var name by mutableStateOf("")
    var assetType by mutableStateOf("CRYPTO")     // CRYPTO | GOLD | STOCK | MUTUAL_FUND
    var symbol by mutableStateOf("")              // opsional
    var units by mutableStateOf("")               // desimal (mis. "1.5") — hanya mode tambah
    var avgBuyPrice by mutableStateOf("")         // rupiah bulat — hanya mode tambah
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    init {
        if (editId != null) {
            viewModelScope.launch {
                repo.refresh()
                repo.find(editId)?.let { inv ->
                    name = inv.name
                    assetType = inv.assetType
                    symbol = inv.symbol ?: ""
                } ?: run { error = "Instrumen tidak ditemukan." }
            }
        }
    }

    private val unitsValue: Double?
        get() = units.trim().toDoubleOrNull()

    private val avgBuyPriceValue: Long?
        get() = avgBuyPrice.filter { it.isDigit() }.toLongOrNull()

    val canSave: Boolean
        get() = !saving && name.isNotBlank() &&
            (isEdit || ((unitsValue ?: 0.0) > 0.0 && (avgBuyPriceValue ?: 0) > 0))

    fun save(onDone: () -> Unit) {
        if (saving || name.isBlank()) {
            error = "Isi nama aset."
            return
        }
        saving = true
        error = null
        viewModelScope.launch {
            try {
                if (editId != null) {
                    repo.update(
                        id = editId,
                        name = name.trim(),
                        assetType = assetType,
                        symbol = symbol,
                    )
                } else {
                    val u = unitsValue
                    val price = avgBuyPriceValue
                    if (u == null || u <= 0.0 || price == null || price <= 0) {
                        saving = false
                        error = "Isi jumlah unit dan harga beli."
                        return@launch
                    }
                    repo.create(
                        name = name.trim(),
                        assetType = assetType,
                        units = u,
                        avgBuyPriceIdr = price,
                        symbol = symbol,
                    )
                }
                saving = false
                onDone()
            } catch (e: Exception) {
                saving = false
                error = "Gagal menyimpan investasi."
            }
        }
    }

    companion object {
        fun factory(
            repo: InvestmentsRepository,
            editId: String? = null,
        ) = viewModelFactory {
            initializer { AddInvestmentViewModel(repo, editId) }
        }
    }
}
