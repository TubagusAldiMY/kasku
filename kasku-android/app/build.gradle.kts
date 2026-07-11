plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "tech.tubsamy.kasku"
    compileSdk = 36

    defaultConfig {
        applicationId = "tech.tubsamy.kasku"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "0.1.0"

        // OAuth Web client ID (server client id untuk Credential Manager). Bukan rahasia —
        // sama dengan PUBLIC_GOOGLE_CLIENT_ID di web. Client SECRET tak dipakai di native.
        buildConfigField(
            "String",
            "GOOGLE_SERVER_CLIENT_ID",
            "\"99280146239-g42tn7h6hbctde310o823cfv0s9ft9kl.apps.googleusercontent.com\"",
        )
    }

    buildTypes {
        debug {
            // Dev: HP akses backend lokal via `adb reverse tcp:18080 tcp:18080`.
            buildConfigField("String", "BASE_URL", "\"http://localhost:18080/v1/\"")
        }
        release {
            isMinifyEnabled = false
            buildConfigField("String", "BASE_URL", "\"https://api-kasku.tubsamy.dev/v1/\"")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlin {
        jvmToolchain(17)
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }
}

dependencies {
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.navigation.compose)
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.ui)
    implementation(libs.androidx.ui.graphics)
    implementation(libs.androidx.ui.tooling.preview)
    implementation(libs.androidx.material3)
    debugImplementation(libs.androidx.ui.tooling)

    // Networking + persistence
    implementation(libs.retrofit)
    implementation(libs.retrofit.kotlinx.serialization)
    implementation(libs.okhttp.logging)
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.androidx.datastore.preferences)
    implementation(libs.androidx.credentials)
    implementation(libs.androidx.credentials.play.services)
    implementation(libs.googleid)

    testImplementation(libs.junit)
    testImplementation(libs.kotlinx.serialization.json)
    testImplementation(libs.retrofit)
    testImplementation(libs.retrofit.kotlinx.serialization)
    testImplementation(libs.okhttp.mockwebserver)
}
