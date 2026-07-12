package tech.tubsamy.kasku.ui.auth

import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import tech.tubsamy.kasku.R
import tech.tubsamy.kasku.ui.components.PrimaryButton
import tech.tubsamy.kasku.ui.theme.KasKuInk
import tech.tubsamy.kasku.ui.theme.KasKuTealSoft

@Composable
fun LoginScreen(
    vm: LoginViewModel,
    onLoggedIn: () -> Unit,
    onRegister: () -> Unit,
) {
    val context = LocalContext.current
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(24.dp),
        verticalArrangement = Arrangement.Center,
    ) {
        // Panel brand ink — adaptasi split panel desain login (ReDesign/), ramping.
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .clip(MaterialTheme.shapes.large)
                .background(KasKuInk)
                .padding(22.dp),
        ) {
            Text(
                buildAnnotatedString {
                    append("Kas")
                    withStyle(SpanStyle(color = KasKuTealSoft, fontStyle = FontStyle.Italic)) { append("Ku") }
                },
                style = MaterialTheme.typography.titleLarge,
                color = MaterialTheme.colorScheme.background,
            )
            Spacer(Modifier.height(14.dp))
            Text(
                "“Kamu tidak perlu jadi kaya untuk mulai tertib. Kamu perlu tertib untuk mulai kaya.”",
                style = MaterialTheme.typography.titleLarge,
                color = MaterialTheme.colorScheme.background,
            )
            Spacer(Modifier.height(10.dp))
            Text(
                "Catatanmu aman, offline maupun online.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.background.copy(alpha = 0.55f),
            )
        }

        Spacer(Modifier.height(28.dp))

        Text("Selamat datang kembali", style = MaterialTheme.typography.headlineMedium)
        Spacer(Modifier.height(6.dp))
        Text(
            "Masuk untuk melanjutkan pencatatan.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        Spacer(Modifier.height(24.dp))

        OutlinedTextField(
            value = vm.email,
            onValueChange = { vm.email = it },
            label = { Text("Email") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email, imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(12.dp))

        OutlinedTextField(
            value = vm.password,
            onValueChange = { vm.password = it },
            label = { Text("Kata sandi") },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
            modifier = Modifier.fillMaxWidth(),
        )

        if (vm.error != null) {
            Spacer(Modifier.height(12.dp))
            Text(
                text = vm.error!!,
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodySmall,
            )
        }

        Spacer(Modifier.height(24.dp))

        PrimaryButton(
            text = "Masuk",
            onClick = { vm.login(onLoggedIn) },
            enabled = vm.email.isNotBlank() && vm.password.isNotBlank(),
            loading = vm.loading,
        )

        Spacer(Modifier.height(12.dp))

        OutlinedButton(
            onClick = { vm.googleSignIn(context, onLoggedIn) },
            enabled = !vm.loading,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Image(
                painterResource(R.drawable.ic_google_g),
                contentDescription = null, // dekoratif — label tombol sudah menjelaskan
                modifier = Modifier.size(16.dp),
            )
            Spacer(Modifier.width(10.dp))
            Text("Masuk dengan Google")
        }

        Spacer(Modifier.height(16.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                "Belum punya akun?",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            TextButton(onClick = onRegister) { Text("Daftar gratis") }
        }
    }
}
