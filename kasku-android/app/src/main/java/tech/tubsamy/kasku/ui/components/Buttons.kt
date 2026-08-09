package tech.tubsamy.kasku.ui.components

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

/**
 * Tombol utama KasKu — CTA teal solid (padanan `bg-teal text-card` di web).
 *
 * State disabled sengaja tetap bernuansa teal, bukan abu-abu default M3 yang di kanvas
 * gelap terbaca seperti field mati. Yang penting: warna LABEL-nya tidak boleh ikut
 * `onPrimary` (= card, hampir hitam) — di atas teal yang sudah diredam jadi gelap, teks
 * gelap praktis hilang. Karena itu label disabled memakai ink diredam supaya kontrasnya
 * tetap terbaca (±4:1).
 */
@Composable
fun PrimaryButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
    loading: Boolean = false,
) {
    Button(
        onClick = onClick,
        enabled = enabled && !loading,
        colors = ButtonDefaults.buttonColors(
            disabledContainerColor = MaterialTheme.colorScheme.primary.copy(alpha = 0.22f),
            disabledContentColor = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.5f),
        ),
        modifier = modifier.fillMaxWidth(),
    ) {
        if (loading) {
            CircularProgressIndicator(
                modifier = Modifier.height(18.dp),
                strokeWidth = 2.dp,
                color = MaterialTheme.colorScheme.onPrimary,
            )
        } else {
            Text(text)
        }
    }
}
