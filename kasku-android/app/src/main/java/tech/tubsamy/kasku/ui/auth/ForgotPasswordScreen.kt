package tech.tubsamy.kasku.ui.auth

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.ui.components.PrimaryButton

@Composable
fun ForgotPasswordScreen(
    vm: ForgotPasswordViewModel,
    onBack: () -> Unit,
    onHaveCode: () -> Unit,
    onVerifyEmail: () -> Unit,
) {
    // Permintaan terkirim → tampilkan konfirmasi + jalan ke layar reset (masukkan kode).
    if (vm.successMessage != null) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(24.dp),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text("Periksa email Anda", style = MaterialTheme.typography.headlineMedium)
            Spacer(Modifier.height(8.dp))
            Text(
                vm.successMessage!!,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(24.dp))
            PrimaryButton(text = "Saya sudah punya kode reset", onClick = onHaveCode)
            Spacer(Modifier.height(8.dp))
            TextButton(onClick = onBack) { Text("Kembali ke masuk") }
        }
        return
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        verticalArrangement = Arrangement.Center,
    ) {
        Text("Lupa password?", style = MaterialTheme.typography.headlineLarge)
        Spacer(Modifier.height(6.dp))
        Text(
            "Masukkan email akunmu. Kami kirim instruksi untuk mereset password.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        Spacer(Modifier.height(24.dp))

        OutlinedTextField(
            value = vm.email,
            onValueChange = { vm.email = it },
            label = { Text("Email") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email, imeAction = ImeAction.Done),
            modifier = Modifier.fillMaxWidth(),
        )

        if (vm.error != null) {
            Spacer(Modifier.height(12.dp))
            Text(vm.error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
        }

        Spacer(Modifier.height(24.dp))

        PrimaryButton(
            text = "Kirim instruksi reset",
            onClick = { vm.submit() },
            enabled = vm.canSubmit,
            loading = vm.loading,
        )

        Spacer(Modifier.height(8.dp))

        TextButton(
            onClick = onHaveCode,
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Sudah punya kode reset?") }

        TextButton(
            onClick = onVerifyEmail,
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Perlu verifikasi email?") }

        TextButton(
            onClick = onBack,
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Kembali ke masuk") }
    }
}
