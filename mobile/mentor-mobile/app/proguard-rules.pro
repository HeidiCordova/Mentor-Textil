# GStreamer
-keep class org.freedesktop.gstreamer.** { *; }
-keepclasseswithmembernames class org.freedesktop.gstreamer.** { *; }

# WebRTC
-keep class org.webrtc.** { *; }
-keepclasseswithmembernames class org.webrtc.** { *; }

# Kotlin Coroutines
-keepclasseswithmembernames class kotlinx.coroutines.** { *; }

# Timber
-dontwarn timber.log.Timber
-keep class timber.log.Timber { *; }

# OkHttp
-dontwarn okhttp3.**
-dontwarn okio.**
-keep class okhttp3.** { *; }
-keep class okio.** { *; }

# GSON
-keep class com.google.gson.** { *; }
-keepclasseswithmembernames class com.google.gson.** { *; }
