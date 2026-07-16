package tech.tubsamy.kasku.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.outlined.Assessment
import androidx.compose.material.icons.outlined.CardMembership
import androidx.compose.material.icons.outlined.ChevronRight
import androidx.compose.material.icons.outlined.Contactless
import androidx.compose.material.icons.outlined.DeleteOutline
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.SwapHoriz
import androidx.compose.material.icons.outlined.Warning
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import tech.tubsamy.kasku.data.AccountItem
import tech.tubsamy.kasku.ui.account.accountTypeLabel
import tech.tubsamy.kasku.ui.components.Hairline
import tech.tubsamy.kasku.ui.components.MoneyText
import tech.tubsamy.kasku.ui.components.SectionLabel
import tech.tubsamy.kasku.ui.theme.KasKuGold
import tech.tubsamy.kasku.ui.theme.KasKuInk
import tech.tubsamy.kasku.ui.theme.KasKuTealSoft
import tech.tubsamy.kasku.ui.theme.LabelEyebrow

@Composable
fun HomeScreen(
    vm: HomeViewModel,
    onAddAccount: () -> Unit,
    onEditAccount: (String) -> Unit,
    onLogout: () -> Unit,
    onConflicts: () -> Unit,
    onDebts: () -> Unit,
    onReports: () -> Unit,
    onBilling: () -> Unit,
    onProfile: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accounts by vm.accounts.collectAsState()
    val conflictCount by vm.conflictCount.collectAsState()
    // Konfirmasi hapus (aksi destruktif — cegah tap tak sengaja).
    var pendingDelete by remember { mutableStateOf<AccountItem?>(null) }

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

        // Satu LazyColumn untuk seluruh sisa layar (Rekening + Lainnya) — semua konten
        // selalu bisa di-scroll sampai terlihat, tak ada bagian yang "kepotong" fillMaxSize.
        LazyColumn(contentPadding = PaddingValues(bottom = 88.dp)) {
            item {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    SectionLabel("Rekening")
                    TextButton(onClick = onAddAccount) {
                        Icon(Icons.Filled.Add, contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.width(4.dp))
                        Text("Tambah")
                    }
                }
                Spacer(Modifier.height(8.dp))
            }

            if (accounts.isEmpty()) {
                item {
                    Column(
                        Modifier.fillMaxWidth(),
                        horizontalAlignment = Alignment.CenterHorizontally,
                    ) {
                        Spacer(Modifier.height(24.dp))
                        Text(
                            "Belum ada akun.",
                            color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.6f),
                        )
                        Spacer(Modifier.height(12.dp))
                        OutlinedButton(onClick = onAddAccount) { Text("Tambah akun") }
                        Spacer(Modifier.height(8.dp))
                        TextButton(onClick = { vm.refresh() }) { Text("Segarkan") }
                        Spacer(Modifier.height(24.dp))
                    }
                }
            } else {
                items(accounts) { account ->
                    AccountRow(
                        account = account,
                        onEdit = { onEditAccount(account.id) },
                        onDelete = { pendingDelete = account },
                    )
                    Hairline()
                }
            }

            item {
                Spacer(Modifier.height(24.dp))
                SectionLabel("Lainnya")
                Spacer(Modifier.height(8.dp))
                LainnyaRow(icon = Icons.Outlined.SwapHoriz, label = "Hutang & Piutang", onClick = onDebts)
                Hairline()
                LainnyaRow(icon = Icons.Outlined.Assessment, label = "Laporan", onClick = onReports)
                Hairline()
                LainnyaRow(icon = Icons.Outlined.CardMembership, label = "Paket", onClick = onBilling)
                Hairline()
                LainnyaRow(icon = Icons.Outlined.Person, label = "Profil", onClick = onProfile)
                Hairline()
            }
        }
    }

    // Dialog konfirmasi hapus.
    pendingDelete?.let { target ->
        AlertDialog(
            onDismissRequest = { pendingDelete = null },
            title = { Text("Hapus akun?") },
            text = { Text("\"${target.name}\" akan dihapus. Tindakan ini tidak bisa dibatalkan.") },
            confirmButton = {
                TextButton(onClick = {
                    vm.deleteAccount(target.id)
                    pendingDelete = null
                }) {
                    Text("Hapus", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingDelete = null }) { Text("Batal") }
            },
        )
    }
}

/** Baris navigasi "Lainnya" (Hutang & Piutang / Laporan / Paket / Profil) — ikon + label + chevron. */
@Composable
private fun LainnyaRow(
    icon: ImageVector,
    label: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 14.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(icon, contentDescription = null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.width(16.dp))
            Text(label, style = MaterialTheme.typography.bodyLarge)
        }
        Icon(
            Icons.Outlined.ChevronRight,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

/** Baris akun: tap area kiri → edit; ikon hapus di kanan → konfirmasi. */
@Composable
private fun AccountRow(
    account: AccountItem,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Row(
            modifier = Modifier
                .weight(1f)
                .clickable(onClick = onEdit)
                .padding(vertical = 14.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f)) {
                Text(account.name, style = MaterialTheme.typography.bodyLarge)
                Text(
                    accountTypeLabel(account.accountType),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Spacer(Modifier.width(12.dp))
            MoneyText(
                account.balance,
                fontSize = 22,
                // Saldo nol diredam agar akun aktif menonjol.
                color = if (account.balance == 0L) {
                    MaterialTheme.colorScheme.onSurfaceVariant
                } else {
                    MaterialTheme.colorScheme.onBackground
                },
            )
        }
        IconButton(onClick = onDelete) {
            Icon(
                Icons.Outlined.DeleteOutline,
                contentDescription = "Hapus ${account.name}",
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
            )
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
