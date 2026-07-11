package tech.tubsamy.kasku

import android.graphics.Color
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import tech.tubsamy.kasku.ui.theme.KasKuTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // App default gelap → ikon status/nav bar terang.
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.dark(Color.TRANSPARENT),
            navigationBarStyle = SystemBarStyle.dark(Color.TRANSPARENT),
        )
        val container = (application as KasKuApplication).container
        setContent {
            KasKuTheme {
                // AppRoot memiliki Scaffold sendiri (bottom nav bar + inset system bar).
                AppRoot(container)
            }
        }
    }
}
