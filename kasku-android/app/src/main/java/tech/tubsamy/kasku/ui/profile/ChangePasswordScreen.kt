package tech.tubsamy.kasku.ui.profile

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.ui.components.PrimaryButton

@Composable
fun ChangePasswordScreen(
    vm: ChangePasswordViewModel,
    onDone: () -> Unit,
    onCancel: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text("Ubah password", style = MaterialTheme.typography.headlineSmall)
            TextButton(onClick = onCancel) { Text("Batal") }
        }

        Spacer(Modifier.height(20.dp))

        OutlinedTextField(
            value = vm.currentPassword,
            onValueChange = { vm.currentPassword = it },
            label = { Text("Password saat ini") },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))

        OutlinedTextField(
            value = vm.newPassword,
            onValueChange = { vm.newPassword = it },
            label = { Text("Password baru") },
            supportingText = { Text("Min. 8 karakter, ada huruf besar, kecil, dan angka.") },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(16.dp))

        OutlinedTextField(
            value = vm.confirmPassword,
            onValueChange = { vm.confirmPassword = it },
            label = { Text("Konfirmasi password baru") },
            isError = vm.confirmPassword.isNotEmpty() && vm.confirmPassword != vm.newPassword,
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            modifier = Modifier.fillMaxWidth(),
        )

        if (vm.error != null) {
            Spacer(Modifier.height(12.dp))
            Text(vm.error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
        }

        Spacer(Modifier.height(24.dp))

        PrimaryButton(
            text = "Simpan",
            onClick = { vm.save(onDone) },
            enabled = vm.canSave,
            loading = vm.saving,
        )
    }
}
