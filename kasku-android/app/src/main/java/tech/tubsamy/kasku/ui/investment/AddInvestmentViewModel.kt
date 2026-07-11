package tech.tubsamy.kasku.ui.investment

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.sync.InvestmentMutations

class AddInvestmentViewModel(
    private val mutations: InvestmentMutations,
) : ViewModel() {

    var name by mutableStateOf("")
    var assetType by mutableStateOf("CRYPTO")     // CRYPTO | GOLD | STOCK | MUTUAL_FUND
    var symbol by mutableStateOf("")              // opsional
    var units by mutableStateOf("")               // desimal (mis. "1.5")
    var avgBuyPrice by mutableStateOf("")         // rupiah bulat (string angka)
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    private val unitsValue: Double?
        get() = units.trim().toDoubleOrNull()

    private val avgBuyPriceValue: Long?
        get() = avgBuyPrice.filter { it.isDigit() }.toLongOrNull()

    val canSave: Boolean
        get() = !saving &&
            name.isNotBlank() &&
            (unitsValue ?: 0.0) > 0.0 &&
            (avgBuyPriceValue ?: 0) > 0

    fun save(onDone: () -> Unit) {
        val u = unitsValue
        val price = avgBuyPriceValue
        if (saving || name.isBlank() || u == null || u <= 0.0 || price == null || price <= 0) {
            error = "Isi nama, jumlah unit, dan harga beli."
            return
        }
        saving = true
        error = null
        viewModelScope.launch {
            try {
                mutations.create(
                    name = name.trim(),
                    assetType = assetType,
                    units = u,
                    avgBuyPriceIdr = price,
                    symbol = symbol,
                )
                saving = false
                onDone() // optimistic: masuk Room + antre sync, langsung kembali
            } catch (e: Exception) {
                saving = false
                error = "Gagal menyimpan investasi."
            }
        }
    }

    companion object {
        fun factory(mutations: InvestmentMutations) = viewModelFactory {
            initializer { AddInvestmentViewModel(mutations) }
        }
    }
}
