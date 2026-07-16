package tech.tubsamy.kasku.ui.debt

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import kotlinx.coroutines.launch
import tech.tubsamy.kasku.data.DebtsRepository

/**
 * Form tambah/edit catatan hutang/piutang. editId != null → prefill dari cache repo.
 * Backend hanya izinkan edit person_name/due_date/notes → saat edit, arah & nominal
 * ditampilkan read-only dan tidak dikirim.
 */
class AddDebtViewModel(
    private val debts: DebtsRepository,
    private val editId: String? = null,
) : ViewModel() {

    val isEdit: Boolean get() = editId != null

    var direction by mutableStateOf("PAYABLE") // PAYABLE (hutang) | RECEIVABLE (piutang)
    var personName by mutableStateOf("")
    var amount by mutableStateOf("") // digit string, rupiah
    var dueDate by mutableStateOf("") // "YYYY-MM-DD" atau kosong
    var notes by mutableStateOf("")
    var saving by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    val canSave: Boolean
        get() = !saving && personName.isNotBlank() && (isEdit || (amount.toLongOrNull() ?: 0L) > 0L)

    init {
        editId?.let { id ->
            debts.debts.value.firstOrNull { it.id == id }?.let { d ->
                direction = d.direction
                personName = d.personName
                amount = d.totalAmount.toString()
                dueDate = d.dueDate ?: ""
                notes = d.notes
            } ?: run { error = "Catatan tidak ditemukan." }
        }
    }

    fun save(onSaved: () -> Unit) {
        if (!canSave) return
        saving = true
        error = null
        viewModelScope.launch {
            try {
                val due = dueDate.ifBlank { null }
                if (editId != null) {
                    debts.update(editId, personName, due, notes)
                } else {
                    debts.create(direction, personName, amount.toLongOrNull() ?: 0L, due, notes)
                }
                saving = false
                onSaved()
            } catch (e: Exception) {
                saving = false
                error = "Gagal menyimpan catatan. Periksa koneksi."
            }
        }
    }

    companion object {
        fun factory(
            debts: DebtsRepository,
            editId: String? = null,
        ) = viewModelFactory {
            initializer { AddDebtViewModel(debts, editId) }
        }
    }
}
