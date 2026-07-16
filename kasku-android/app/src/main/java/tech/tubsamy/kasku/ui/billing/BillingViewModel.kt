package tech.tubsamy.kasku.ui.billing

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.BillingRepository
import tech.tubsamy.kasku.data.remote.dto.PlanDto
import tech.tubsamy.kasku.data.remote.dto.SubscribeResponseDto
import tech.tubsamy.kasku.data.remote.dto.SubscriptionDto

/** Layar langganan: daftar plan + status langganan aktif + inisiasi pembayaran. */
class BillingViewModel(
    private val repo: BillingRepository,
) : ViewModel() {

    var plans by mutableStateOf<List<PlanDto>>(emptyList())
        private set
    var subscription by mutableStateOf<SubscriptionDto?>(null)
        private set
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    // Info pembayaran hasil subscribe (tampilkan dialog/URL). null = belum ada.
    var payment by mutableStateOf<SubscribeResponseDto?>(null)
        private set
    // plan_id yang sedang diproses subscribe (untuk spinner per-tombol).
    var subscribingPlanId by mutableStateOf<String?>(null)
        private set

    init {
        load()
    }

    fun load() {
        loading = true
        error = null
        viewModelScope.launch {
            try {
                plans = repo.plans()
                subscription = repo.subscription()
            } catch (e: Exception) {
                error = "Gagal memuat langganan. Periksa koneksi."
            } finally {
                loading = false
            }
        }
    }

    fun subscribe(planId: String) {
        if (subscribingPlanId != null) return
        subscribingPlanId = planId
        error = null
        viewModelScope.launch {
            try {
                payment = repo.subscribe(planId)
            } catch (e: Exception) {
                error = "Gagal memulai pembayaran. Coba lagi."
            } finally {
                subscribingPlanId = null
            }
        }
    }

    fun dismissPayment() {
        payment = null
    }

    companion object {
        fun factory(repo: BillingRepository) = viewModelFactory {
            initializer { BillingViewModel(repo) }
        }
    }
}
