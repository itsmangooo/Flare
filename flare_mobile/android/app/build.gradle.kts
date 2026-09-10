plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

val releaseSigningEnvironment = mapOf(
    "FLARE_ANDROID_KEYSTORE" to System.getenv("FLARE_ANDROID_KEYSTORE"),
    "FLARE_ANDROID_KEY_ALIAS" to System.getenv("FLARE_ANDROID_KEY_ALIAS"),
    "FLARE_ANDROID_KEYSTORE_PASSWORD" to System.getenv("FLARE_ANDROID_KEYSTORE_PASSWORD"),
    "FLARE_ANDROID_KEY_PASSWORD" to System.getenv("FLARE_ANDROID_KEY_PASSWORD"),
)
val missingReleaseSigningValues = releaseSigningEnvironment.filterValues { it.isNullOrBlank() }.keys
val releaseBuildRequested = gradle.startParameter.taskNames.any { it.contains("release", ignoreCase = true) }
if (releaseBuildRequested && missingReleaseSigningValues.isNotEmpty()) {
    throw GradleException(
        "Flare Release signing is not configured. Missing: ${missingReleaseSigningValues.joinToString()}. " +
            "Set all FLARE_ANDROID_* environment variables; Release builds never use the debug key.",
    )
}

android {
    namespace = "io.github.itsmangooo.flare"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        applicationId = "io.github.itsmangooo.flare"
        minSdk = 26
        targetSdk = flutter.targetSdkVersion
        // Uses the version code from pubspec.yaml. When using split APKs, 1000 * ABI_VERSION
        // is added automatically by Flutter. (https://developer.android.com/studio/build/configure-apk-splits#configure-APK-versions)
        // You can force using the value of versionCode by specifying the `-P force-version-code-ignoring-abi=true`
        // flag during build.
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    signingConfigs {
        if (missingReleaseSigningValues.isEmpty()) {
            create("release") {
                storeFile = file(releaseSigningEnvironment.getValue("FLARE_ANDROID_KEYSTORE")!!)
                keyAlias = releaseSigningEnvironment.getValue("FLARE_ANDROID_KEY_ALIAS")
                storePassword = releaseSigningEnvironment.getValue("FLARE_ANDROID_KEYSTORE_PASSWORD")
                keyPassword = releaseSigningEnvironment.getValue("FLARE_ANDROID_KEY_PASSWORD")
            }
        }
    }

    buildTypes {
        release {
            if (missingReleaseSigningValues.isEmpty()) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}
