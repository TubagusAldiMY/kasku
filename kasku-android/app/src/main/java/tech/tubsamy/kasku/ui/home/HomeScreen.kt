package tech.tubsamy.kasku.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Logout
import androidx.compose.material.icons.outlined.Contactless
import androidx.compose.material.icons.outlined.Warning
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.components.LedgerRow
import tech.tubsamy.kasku.ui.components.MoneyText
import tech.tubsamy.kasku.ui.components.SectionLabel
import tech.tubsamy.kasku.ui.theme.KasKuGold
import tech.tubsamy.kasku.ui.theme.KasKuInk
import tech.tubsamy.kasku.ui.theme.KasKuTealSoft
import tech.tubsamy.kasku.ui.theme.LabelEyebrow

@Composable
fun HomeScreen(
    vm: HomeViewModel,
    onLogout: () -> Unit,
    onConflicts: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accounts by vm.accounts.collectAsState()
    val conflictCount by vm.conflictCount.collectAsState()

    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(24.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "Akun saya",
                style = MaterialTheme.typography.headlineMedium,
            )
            Row(verticalAlignment = Alignment.CenterVertically) {
                // Konflik hanya muncul saat ada — badge coral, tap → layar konflik.
                if (conflictCount > 0) {
                    TextButton(onClick = onConflicts) {
                        Icon(
                            Icons.Outlined.Warning,
                            contentDescription = "Konflik",
                            tint = MaterialTheme.colorScheme.error,
                            modifier = Modifier.size(18.dp),
                        )
                        Spacer(Modifier.width(4.dp))
                        Text("$conflictCount", color = MaterialTheme.colorScheme.error)
                    }
                }
                IconButton(onClick = onLogout) {
                    Icon(
                        Icons.AutoMirrored.Outlined.Logout,
                        contentDescription = "Keluar",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        Spacer(Modifier.height(20.dp))

        BalanceCard(total = accounts.sumOf { it.balance })

        Spacer(Modifier.height(24.dp))
        SectionLabel("Rekening")
        Spacer(Modifier.height(8.dp))

        if (accounts.isEmpty()) {
            Column(
                Modifier.fillMaxSize(),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    "Belum ada akun.",
                    color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.6f),
                )
                Spacer(Modifier.height(12.dp))
                OutlinedButton(onClick = { vm.refresh() }) { Text("Segarkan") }
            }
        } else {
            LazyColumn(contentPadding = PaddingValues(bottom = 88.dp)) {
                items(accounts) { account ->
                    LedgerRow(
                        label = account.name,
                        amount = account.balance,
                        sub = account.accountType,
                        // Saldo nol diredam agar akun aktif menonjol.
                        color = if (account.balance == 0L) {
                            MaterialTheme.colorScheme.onSurfaceVariant
                        } else {
                            MaterialTheme.colorScheme.onBackground
                        },
                    )
                    Hairline()
                }
            }
        }
    }
}

/** Kartu saldo bergaya kartu fisik di atas ink brand — chip emas, contactless, nomor bertopeng. */
@Composable
private fun BalanceCard(total: Long) {
    val paper = MaterialTheme.colorScheme.background
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(200.dp)
            .clip(RoundedCornerShape(20.dp))
            .background(KasKuInk)
            .padding(24.dp),
    ) {
        Column(
            modifier = Modifier.fillMaxSize(),
            verticalArrangement = Arrangement.SpaceBetween,
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                // Chip emas EMV — gradasi gold palet editorial.
                Box(
                    modifier = Modifier
                        .size(width = 46.dp, height = 34.dp)
                        .clip(RoundedCornerShape(7.dp))
                        .background(
                            Brush.linearGradient(colors = listOf(Color(0xFFD9BC6A), KasKuGold)),
                        ),
                )
                Icon(
                    Icons.Outlined.Contactless,
                    contentDescription = null,
                    tint = paper.copy(alpha = 0.85f),
                )
            }
            Column {
                Text(
                    "TOTAL SALDO",
                    style = LabelEyebrow,
                    color = paper.copy(alpha = 0.7f),
                )
                Spacer(Modifier.height(2.dp))
                MoneyText(total, fontSize = 34, color = paper)
            }
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    "•••• •••• •••• ••••",
                    modifier = Modifier.weight(1f),
                    color = paper.copy(alpha = 0.75f),
                    fontSize = 15.sp,
                    letterSpacing = 1.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Clip,
                )
                Spacer(Modifier.width(12.dp))
                // Wordmark editorial: "Kas" + "Ku" italic teal lembut.
                Text(
                    buildAnnotatedString {
                        append("Kas")
                        withStyle(SpanStyle(color = KasKuTealSoft, fontStyle = FontStyle.Italic)) { append("Ku") }
                    },
                    style = MaterialTheme.typography.titleLarge,
                    color = paper,
                    maxLines = 1,
                    softWrap = false,
                )
            }
        }
    }
}
