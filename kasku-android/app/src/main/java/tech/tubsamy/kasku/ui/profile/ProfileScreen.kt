package tech.tubsamy.kasku.ui.profile

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.data.remote.dto.ProfileDto

@Composable
fun ProfileScreen(
    vm: ProfileViewModel,
    onChangePassword: () -> Unit,
    onLogout: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val profile = vm.profile

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(24.dp),
    ) {
        Text("Profil", style = MaterialTheme.typography.headlineMedium)
        Spacer(Modifier.height(16.dp))

        when {
            vm.error != null -> {
                Text(vm.error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                Spacer(Modifier.height(12.dp))
                OutlinedButton(onClick = { vm.load() }) { Text("Coba lagi") }
            }

            vm.loading && profile == null -> {
                Row(Modifier.fillMaxWidth().padding(top = 16.dp), horizontalArrangement = Arrangement.Center) {
                    CircularProgressIndicator()
                }
            }

            profile != null -> ProfileBody(profile)
        }

        Spacer(Modifier.height(24.dp))
        OutlinedButton(onClick = onChangePassword, modifier = Modifier.fillMaxWidth()) {
            Text("Ubah password")
        }
        Spacer(Modifier.height(12.dp))
        TextButton(onClick = onLogout, modifier = Modifier.fillMaxWidth()) {
            Text("Keluar", color = MaterialTheme.colorScheme.error)
        }
    }
}

@Composable
private fun ProfileBody(profile: ProfileDto) {
    Column {
        InfoRow("Nama", profile.displayName?.takeIf { it.isNotBlank() } ?: profile.username.ifBlank { "—" })
        HorizontalDivider()
        InfoRow("Email", profile.email.ifBlank { "—" })
        HorizontalDivider()
        InfoRow("Paket", tierLabel(profile.subscriptionTier))
    }
}

@Composable
private fun InfoRow(label: String, value: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 14.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.Medium)
    }
}

private fun tierLabel(tier: String): String = when (tier.lowercase()) {
    "free" -> "Gratis"
    "premium" -> "Premium"
    "pro" -> "Pro"
    "" -> "—"
    else -> tier
}
