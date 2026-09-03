# Flare Liquid Glass renderer

Flare's glass controls are MAUI views backed by `AndroidLiquidGlassView`. The native view captures only its own on-screen backdrop region when it attaches or changes size. That bitmap is recorded into a cached `RenderNode`; there is no continuous full-screen capture or animation timer.

Rendering tiers:

- Android 13 / API 33 and newer: an AGSL `RuntimeShader` refracts the captured pixels, splits red/blue samples near the edge, applies tint, edge/specular light, touch response, inner depth, and then chains the shader over a `RenderEffect` blur.
- Android 12 / API 31-32: the cached backdrop uses `RenderEffect` blur with cached native gradient highlights, tint, edge color separation, and depth.
- Android API 26-30: the same native shape and cached layered highlights render without unsupported effect APIs.

Every advanced API call is both version-gated and protected by runtime fallback handling. A GPU/driver shader failure falls back to the next supported tier instead of failing view creation or terminating the app.

In Debug builds, a tier can be forced for device verification without changing production behavior:

```powershell
$activity = (adb shell cmd package resolve-activity --brief io.github.itsmangooo.flare | Select-Object -Last 1).Trim()
adb shell am start -n $activity --es flare.glass.tier layered
adb shell am start -n $activity --es flare.glass.tier renderEffect
```

The override is compiled out of Release builds.
