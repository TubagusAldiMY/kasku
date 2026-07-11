package tech.tubsamy.kasku

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.ui.Modifier
import tech.tubsamy.kasku.ui.theme.KasKuTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        val container = (application as KasKuApplication).container
        setContent {
            KasKuTheme {
                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    // Inset semua layar dari status bar (judul tak nyelip) & nav bar (FAB tak ketutup).
                    Box(Modifier.padding(innerPadding)) {
                        AppRoot(container)
                    }
                }
            }
        }
    }
}
