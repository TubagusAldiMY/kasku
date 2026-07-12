package tech.tubsamy.kasku.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.BudgetItem
import tech.tubsamy.kasku.ui.formatIdr
import tech.tubsamy.kasku.ui.theme.KasKuGold

/**
 * Blok progres anggaran (desain ReDesign/): nama + persen berwarna, bar tipis 3dp,
 * caption terpakai/limit/sisa. Warna: lewat limit = clay, ≥ ambang alert = gold, aman = teal.
 * Dipakai Dashboard (read-only) & layar kelola anggaran (dibungkus aksi edit/hapus).
 */
@Composable
fun BudgetProgress(b: BudgetItem, modifier: Modifier = Modifier) {
    val accent = when {
        b.isOverBudget -> MaterialTheme.colorScheme.error
        b.alertThreshold in 1..b.progressPercent -> KasKuGold
        else -> MaterialTheme.colorScheme.primary
    }
    Column(modifier.fillMaxWidth().padding(vertical = 12.dp)) {
        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(b.name, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.Medium)
            Text(
                "${b.progressPercent}%",
                style = MaterialTheme.typography.titleSmall,
                color = accent,
            )
        }
        Spacer(Modifier.height(8.dp))
        // Bar progres editorial 3dp — track hairline, isi warna status.
        Box(
            Modifier
                .fillMaxWidth()
                .height(3.dp)
                .background(MaterialTheme.colorScheme.outlineVariant),
        ) {
            Box(
                Modifier
                    .fillMaxWidth((b.progressPercent / 100f).coerceIn(0f, 1f))
                    .height(3.dp)
                    .background(accent),
            )
        }
        Spacer(Modifier.height(8.dp))
        Text(
            "${formatIdr(b.spentIdr)} terpakai · limit ${formatIdr(b.limitIdr)} · " +
                if (b.isOverBudget) "lewat ${formatIdr(-b.remainingIdr)}" else "sisa ${formatIdr(b.remainingIdr)}",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}
