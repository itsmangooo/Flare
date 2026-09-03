using Android.Graphics;
using Android.OS;
using Android.Views;
using System.Runtime.Versioning;
using Flare.Mobile.Controls;
using Microsoft.Maui.Handlers;
using Microsoft.Maui.Platform;
using AColor = Android.Graphics.Color;
using APaint = Android.Graphics.Paint;
using APath = Android.Graphics.Path;
using ARect = Android.Graphics.Rect;
using AView = Android.Views.View;

namespace Flare.Mobile.Platforms.Android.LiquidGlass;

public sealed class LiquidGlassSurfaceHandler : ViewHandler<LiquidGlassSurface, AndroidLiquidGlassView>
{
    public static readonly IPropertyMapper<LiquidGlassSurface, LiquidGlassSurfaceHandler> Mapper =
        new PropertyMapper<LiquidGlassSurface, LiquidGlassSurfaceHandler>(ViewMapper)
        {
            [nameof(LiquidGlassSurface.BlurRadius)] = MapAppearance,
            [nameof(LiquidGlassSurface.RefractionStrength)] = MapAppearance,
            [nameof(LiquidGlassSurface.TintColor)] = MapAppearance,
            [nameof(LiquidGlassSurface.TintOpacity)] = MapAppearance,
            [nameof(LiquidGlassSurface.CornerRadius)] = MapAppearance,
            [nameof(LiquidGlassSurface.HighlightStrength)] = MapAppearance,
            [nameof(LiquidGlassSurface.ChromaticAberration)] = MapAppearance,
            [nameof(LiquidGlassSurface.ShadowStrength)] = MapAppearance,
            [nameof(LiquidGlassSurface.EffectType)] = MapAppearance,
            [nameof(LiquidGlassSurface.TouchX)] = MapInteraction,
            [nameof(LiquidGlassSurface.TouchY)] = MapInteraction,
            [nameof(LiquidGlassSurface.PressedProgress)] = MapInteraction
        };

    public LiquidGlassSurfaceHandler() : base(Mapper)
    {
    }

    protected override AndroidLiquidGlassView CreatePlatformView() => new(Context);

    protected override void ConnectHandler(AndroidLiquidGlassView platformView)
    {
        base.ConnectHandler(platformView);
        platformView.UpdateAppearance(VirtualView);
        platformView.UpdateInteraction(VirtualView);
    }

    private static void MapAppearance(LiquidGlassSurfaceHandler handler, LiquidGlassSurface view) =>
        handler.PlatformView.UpdateAppearance(view);

    private static void MapInteraction(LiquidGlassSurfaceHandler handler, LiquidGlassSurface view) =>
        handler.PlatformView.UpdateInteraction(view);
}

public sealed class AndroidLiquidGlassView : AView
{
    private static readonly float[] BaseGradientStops = [0f, 0.52f, 1f];

    private const string ShaderSource = """
        uniform shader uContent;
        uniform float2 uSize;
        uniform float uRadius;
        uniform float uRefraction;
        uniform float4 uTint;
        uniform float uTintOpacity;
        uniform float uHighlight;
        uniform float uChromatic;
        uniform float uShadow;
        uniform float2 uTouch;
        uniform float uPressed;
        uniform float uClearMode;

        float roundedSdf(float2 p, float2 halfSize, float radius) {
            float2 q = abs(p) - (halfSize - float2(radius));
            return length(max(q, float2(0.0))) + min(max(q.x, q.y), 0.0) - radius;
        }

        half4 main(float2 p) {
            float2 safeSize = max(uSize, float2(1.0));
            float2 uv = p / safeSize;
            float2 centerPx = p - safeSize * 0.5;
            float radius = min(uRadius, min(safeSize.x, safeSize.y) * 0.5);
            float distanceToEdge = roundedSdf(centerPx, safeSize * 0.5, radius);
            float innerDistance = max(0.0, -distanceToEdge);
            float edgeBand = 1.0 - smoothstep(0.0, max(10.0, min(safeSize.x, safeSize.y) * 0.22), innerDistance);

            float2 radial = (uv - 0.5) * float2(safeSize.x / max(safeSize.y, 1.0), 1.0);
            float radialLength = max(length(radial), 0.001);
            float2 normal = radial / radialLength;
            float2 touchDelta = uv - uTouch;
            float touchWave = exp(-dot(touchDelta, touchDelta) * 20.0) * uPressed;
            float lens = edgeBand * (0.58 + 0.42 * uPressed) + touchWave * 0.32;
            float2 sampleOffset = normal * uRefraction * lens + normalize(touchDelta + float2(0.001)) * touchWave * uRefraction * 0.18;

            float chroma = uChromatic * edgeBand;
            half4 baseSample = uContent.eval(clamp(p - sampleOffset, float2(0.0), safeSize - 1.0));
            half red = uContent.eval(clamp(p - sampleOffset - normal * chroma, float2(0.0), safeSize - 1.0)).r;
            half blue = uContent.eval(clamp(p - sampleOffset + normal * chroma, float2(0.0), safeSize - 1.0)).b;
            half3 refracted = half3(red, baseSample.g, blue);

            float directional = clamp(dot(normalize(float2(-0.72, -0.69)), normal), 0.0, 1.0);
            float rim = edgeBand * (0.22 + directional * 0.78);
            float2 specDelta = uv - mix(float2(0.22, 0.12), uTouch, uPressed * 0.7);
            float specular = exp(-dot(specDelta, specDelta) * 36.0) * uHighlight * (0.42 + uPressed * 0.3);
            float lowerDepth = smoothstep(0.48, 1.0, uv.y) * edgeBand * uShadow;
            float clearFactor = mix(1.0, 0.72, uClearMode);

            half3 color = mix(refracted, half3(uTint.rgb), half(uTintOpacity * clearFactor));
            color += half3(1.0, 0.96, 0.93) * half((rim * 0.16 + specular * 0.12) * uHighlight);
            color -= half3(lowerDepth * 0.08);
            color += half3(touchWave * 0.025);
            return half4(color, 1.0);
        }
        """;

    private readonly APaint _bitmapPaint = new(PaintFlags.AntiAlias | PaintFlags.FilterBitmap);
    private readonly APaint _fillPaint = new(PaintFlags.AntiAlias);
    private readonly APaint _strokePaint = new(PaintFlags.AntiAlias) { StrokeWidth = 1, StrokeJoin = APaint.Join.Round };
    private readonly APath _clipPath = new();
    private readonly Matrix _specularMatrix = new();
    private Bitmap? _backdrop;
    private RenderNode? _renderNode;
    private RuntimeShader? _runtimeShader;
    private RenderEffect? _renderEffect;
    private LinearGradient? _baseGradient;
    private RadialGradient? _specularGradient;
    private LinearGradient? _depthGradient;
    private LiquidGlassRenderTier _tier = LiquidGlassRenderTier.LayeredFallback;
    private float _density;
    private float _blurRadius;
    private float _refraction;
    private float _cornerRadius;
    private float _highlight;
    private float _chromatic;
    private float _shadow;
    private float _tintOpacity;
    private float _touchX = 0.25f;
    private float _touchY = 0.15f;
    private float _pressed;
    private AColor _tint = AColor.Rgb(23, 25, 29);
    private bool _clearMode;
    private bool _captureScheduled;
    private bool _effectDirty = true;
    private bool _fallbackShadersDirty = true;
    private bool _capturing;
    private bool _tierReported;

    public AndroidLiquidGlassView(global::Android.Content.Context context) : base(context)
    {
        _density = Resources?.DisplayMetrics?.Density ?? 1;
        SetLayerType(LayerType.Hardware, null);
        SetWillNotDraw(false);
    }

    public void UpdateAppearance(LiquidGlassSurface view)
    {
        _blurRadius = Math.Max(0, (float)view.BlurRadius * _density);
        _refraction = Math.Max(0, (float)view.RefractionStrength * _density);
        _cornerRadius = Math.Max(0, (float)view.CornerRadius * _density);
        _highlight = Math.Clamp((float)view.HighlightStrength, 0, 1.5f);
        _chromatic = Math.Max(0, (float)view.ChromaticAberration * _density);
        _shadow = Math.Clamp((float)view.ShadowStrength, 0, 1.5f);
        _tintOpacity = Math.Clamp((float)view.TintOpacity, 0, 1);
        _clearMode = view.EffectType == LiquidGlassEffectType.Clear;
        _tint = view.TintColor.ToPlatform();
        _effectDirty = true;
        _fallbackShadersDirty = true;
        Invalidate();
    }

    public void UpdateInteraction(LiquidGlassSurface view)
    {
        _touchX = Math.Clamp((float)view.TouchX, 0, 1);
        _touchY = Math.Clamp((float)view.TouchY, 0, 1);
        _pressed = Math.Clamp((float)view.PressedProgress, 0, 1);
        if (OperatingSystem.IsAndroidVersionAtLeast(33)) UpdateShaderUniforms();
        Invalidate();
    }

    protected override void OnAttachedToWindow()
    {
        base.OnAttachedToWindow();
        SelectRenderTier();
        ScheduleBackdropCapture();
    }

    protected override void OnDetachedFromWindow()
    {
        ReleaseGraphicsResources();
        base.OnDetachedFromWindow();
    }

    protected override void OnSizeChanged(int w, int h, int oldw, int oldh)
    {
        base.OnSizeChanged(w, h, oldw, oldh);
        if (w == oldw && h == oldh) return;
        _backdrop?.Dispose();
        _backdrop = null;
        _effectDirty = true;
        _fallbackShadersDirty = true;
        ScheduleBackdropCapture();
    }

    protected override void OnDraw(Canvas canvas)
    {
        base.OnDraw(canvas);
        if (_capturing || Width <= 0 || Height <= 0) return;

        try
        {
            BuildClipPath();
            var save = canvas.Save();
            canvas.ClipPath(_clipPath);
            if (!canvas.IsHardwareAccelerated)
            {
                if (_backdrop is not null && !_backdrop.IsRecycled)
                    canvas.DrawBitmap(_backdrop, null, new ARect(0, 0, Width, Height), _bitmapPaint);
                else
                    DrawFallbackBase(canvas);
                DrawLayeredHighlights(canvas);
                canvas.RestoreToCount(save);
                return;
            }

            if (_backdrop is not null && !_backdrop.IsRecycled)
            {
                DrawBackdrop(canvas);
            }
            else
            {
                DrawFallbackBase(canvas);
            }

            if (_tier != LiquidGlassRenderTier.RuntimeShader)
            {
                DrawLayeredHighlights(canvas);
            }
            canvas.RestoreToCount(save);
        }
        catch (Exception exception)
        {
            global::Android.Util.Log.Warn("FlareLiquidGlass", $"Glass draw failed; using layered fallback. {exception.Message}");
            _tier = LiquidGlassRenderTier.LayeredFallback;
            ReleaseEffects();
            DrawFallbackBase(canvas);
            DrawLayeredHighlights(canvas);
        }
    }

    private void DrawBackdrop(Canvas canvas)
    {
        if (OperatingSystem.IsAndroidVersionAtLeast(31) && BuildRenderNode())
        {
            canvas.DrawRenderNode(_renderNode!);
            return;
        }

        canvas.DrawBitmap(_backdrop!, null, new ARect(0, 0, Width, Height), _bitmapPaint);
    }

    [SupportedOSPlatform("android31.0")]
    private bool BuildRenderNode()
    {
        if (_tier == LiquidGlassRenderTier.LayeredFallback || Build.VERSION.SdkInt < BuildVersionCodes.S) return false;
        if (!_effectDirty && _renderNode is not null) return true;

        try
        {
            _renderNode ??= new RenderNode("FlareLiquidGlassBackdrop");
            _renderNode.SetPosition(0, 0, Width, Height);
            var recording = _renderNode.BeginRecording(Width, Height);
            recording.DrawBitmap(_backdrop!, null, new ARect(0, 0, Width, Height), _bitmapPaint);
            _renderNode.EndRecording();

            ReleaseEffects();
            var blur = RenderEffect.CreateBlurEffect(
                Math.Max(0.1f, _blurRadius),
                Math.Max(0.1f, _blurRadius),
                Shader.TileMode.Clamp!);

            if (_tier == LiquidGlassRenderTier.RuntimeShader && OperatingSystem.IsAndroidVersionAtLeast(33))
            {
                _runtimeShader = new RuntimeShader(ShaderSource);
                UpdateShaderUniforms();
                var glass = RenderEffect.CreateRuntimeShaderEffect(_runtimeShader, "uContent");
                _renderEffect = RenderEffect.CreateChainEffect(glass, blur);
                glass.Dispose();
                blur.Dispose();
                ReportTier("RuntimeShader + chained RenderEffect blur");
            }
            else
            {
                _renderEffect = blur;
                ReportTier("RenderEffect blur + layered tint/highlight");
            }

            _renderNode.SetRenderEffect(_renderEffect);
            _effectDirty = false;
            return true;
        }
        catch (Exception exception)
        {
            global::Android.Util.Log.Warn("FlareLiquidGlass", $"Advanced glass unavailable; degrading safely. {exception.Message}");
            _tier = LiquidGlassFeaturePolicy.Resolve((int)Build.VERSION.SdkInt, runtimeShaderAvailable: false);
            if (_tier == LiquidGlassRenderTier.LayeredFallback) ReportTier("layered compatibility fallback");
            ReleaseEffects();
            _effectDirty = true;
            return _tier == LiquidGlassRenderTier.RenderEffect && BuildRenderNode();
        }
    }

    [SupportedOSPlatform("android33.0")]
    private void UpdateShaderUniforms()
    {
        if (_runtimeShader is null || Width <= 0 || Height <= 0) return;
        try
        {
            _runtimeShader.SetFloatUniform("uSize", Width, Height);
            _runtimeShader.SetFloatUniform("uRadius", _cornerRadius);
            _runtimeShader.SetFloatUniform("uRefraction", _refraction);
            _runtimeShader.SetFloatUniform("uTint", _tint.R / 255f, _tint.G / 255f, _tint.B / 255f, 1f);
            _runtimeShader.SetFloatUniform("uTintOpacity", _tintOpacity);
            _runtimeShader.SetFloatUniform("uHighlight", _highlight);
            _runtimeShader.SetFloatUniform("uChromatic", _chromatic);
            _runtimeShader.SetFloatUniform("uShadow", _shadow);
            _runtimeShader.SetFloatUniform("uTouch", _touchX, _touchY);
            _runtimeShader.SetFloatUniform("uPressed", _pressed);
            _runtimeShader.SetFloatUniform("uClearMode", _clearMode ? 1f : 0f);
        }
        catch (Exception exception)
        {
            global::Android.Util.Log.Warn("FlareLiquidGlass", $"Shader uniform update failed. {exception.Message}");
        }
    }

    private void SelectRenderTier()
    {
        _tier = ResolveRenderTier(Context!);
        if (_tier == LiquidGlassRenderTier.LayeredFallback) ReportTier("layered compatibility fallback");
        _effectDirty = true;
    }

    private static LiquidGlassRenderTier ResolveRenderTier(global::Android.Content.Context initialContext)
    {
#if DEBUG
        var context = initialContext;
        while (context is global::Android.Content.ContextWrapper wrapper && context is not global::Android.App.Activity)
        {
            context = wrapper.BaseContext;
        }

        var forced = (context as global::Android.App.Activity)?.Intent?.GetStringExtra("flare.glass.tier");
        if (string.Equals(forced, "layered", StringComparison.OrdinalIgnoreCase))
            return LiquidGlassRenderTier.LayeredFallback;
        if (string.Equals(forced, "renderEffect", StringComparison.OrdinalIgnoreCase) && OperatingSystem.IsAndroidVersionAtLeast(31))
            return LiquidGlassRenderTier.RenderEffect;
#else
        _ = initialContext;
#endif
        return LiquidGlassFeaturePolicy.Resolve((int)Build.VERSION.SdkInt);
    }

    private void ScheduleBackdropCapture()
    {
        if (_captureScheduled || !IsAttachedToWindow || Width <= 0 || Height <= 0) return;
        _captureScheduled = true;
        Post(() =>
        {
            _captureScheduled = false;
            CaptureBackdrop();
        });
    }

    private void CaptureBackdrop()
    {
        if (!IsAttachedToWindow || Width <= 0 || Height <= 0) return;
        try
        {
            var bitmap = Bitmap.CreateBitmap(Width, Height, Bitmap.Config.Argb8888!);
            using var captureCanvas = new Canvas(bitmap);
            captureCanvas.DrawColor(AColor.Transparent, PorterDuff.Mode.Clear!);

            var root = RootView;
            if (root is null)
            {
                bitmap.Dispose();
                return;
            }

            var viewLocation = new int[2];
            var rootLocation = new int[2];
            GetLocationOnScreen(viewLocation);
            root.GetLocationOnScreen(rootLocation);
            captureCanvas.Translate(rootLocation[0] - viewLocation[0], rootLocation[1] - viewLocation[1]);

            var glassContainer = Parent as AView;
            var previousAlpha = glassContainer?.Alpha ?? 1f;
            try
            {
                _capturing = true;
                if (glassContainer is not null) glassContainer.Alpha = 0;
                root.Draw(captureCanvas);
            }
            finally
            {
                if (glassContainer is not null) glassContainer.Alpha = previousAlpha;
                _capturing = false;
            }

            _backdrop?.Dispose();
            _backdrop = bitmap;
            _effectDirty = true;
            Invalidate();
        }
        catch (Exception exception)
        {
            global::Android.Util.Log.Warn("FlareLiquidGlass", $"Backdrop capture unavailable; using fallback. {exception.Message}");
            _backdrop?.Dispose();
            _backdrop = null;
            Invalidate();
        }
    }

    private void BuildClipPath()
    {
        _clipPath.Reset();
        _clipPath.AddRoundRect(0, 0, Width, Height, _cornerRadius, _cornerRadius, APath.Direction.Cw!);
    }

    private void DrawFallbackBase(Canvas canvas)
    {
        BuildFallbackShaders();
        _fillPaint.Alpha = 255;
        _fillPaint.SetShader(_baseGradient);
        canvas.DrawPath(_clipPath, _fillPaint);
        _fillPaint.SetShader(null);
    }

    private void DrawLayeredHighlights(Canvas canvas)
    {
        BuildFallbackShaders();
        var touchX = Width * _touchX;
        var touchY = Height * _touchY;
        _specularMatrix.Reset();
        _specularMatrix.SetTranslate(touchX, touchY);
        _specularGradient?.SetLocalMatrix(_specularMatrix);
        _fillPaint.Alpha = (int)(255 * Math.Clamp((_highlight * 0.46f) + (_pressed * 0.18f), 0, 0.8f));
        _fillPaint.SetShader(_specularGradient);
        canvas.DrawPath(_clipPath, _fillPaint);
        _fillPaint.SetShader(null);
        _fillPaint.Alpha = 255;

        _strokePaint.SetStyle(APaint.Style.Stroke);
        _strokePaint.StrokeWidth = Math.Max(1, _density * 0.72f);
        _strokePaint.Color = AColor.Argb((int)(255 * Math.Clamp(_highlight * 0.32f, 0, 0.5f)), 245, 238, 233);
        canvas.DrawPath(_clipPath, _strokePaint);

        if (_chromatic > 0.1f)
        {
            var chromaticAlpha = (int)(255 * Math.Clamp(_chromatic / (_density * 20f), 0, 0.11f));
            _strokePaint.Color = AColor.Argb(chromaticAlpha, 255, 88, 68);
            canvas.Save();
            canvas.Translate(-Math.Min(_chromatic, _density * 1.5f), 0);
            canvas.DrawPath(_clipPath, _strokePaint);
            canvas.Restore();
            _strokePaint.Color = AColor.Argb(chromaticAlpha, 74, 174, 255);
            canvas.Save();
            canvas.Translate(Math.Min(_chromatic, _density * 1.5f), 0);
            canvas.DrawPath(_clipPath, _strokePaint);
            canvas.Restore();
        }

        _fillPaint.SetShader(_depthGradient);
        canvas.DrawPath(_clipPath, _fillPaint);
        _fillPaint.SetShader(null);
    }

    private void BuildFallbackShaders()
    {
        if (!_fallbackShadersDirty || Width <= 0 || Height <= 0) return;
        _baseGradient?.Dispose();
        _specularGradient?.Dispose();
        _depthGradient?.Dispose();

        _baseGradient = new LinearGradient(
            0,
            0,
            Width,
            Height,
            new[]
            {
                AColor.Argb((int)(255 * Math.Min(1, _tintOpacity + 0.08f)), _tint.R, _tint.G, _tint.B).ToArgb(),
                AColor.Argb((int)(255 * _tintOpacity), Math.Max(0, _tint.R - 4), Math.Max(0, _tint.G - 4), Math.Max(0, _tint.B - 3)).ToArgb(),
                AColor.Argb((int)(255 * Math.Min(1, _tintOpacity + 0.12f)), 9, 10, 12).ToArgb()
            },
            BaseGradientStops,
            Shader.TileMode.Clamp!);
        _specularGradient = new RadialGradient(
            0,
            0,
            Math.Max(Width, Height) * 0.72f,
            AColor.Argb(92, 255, 244, 238),
            AColor.Transparent,
            Shader.TileMode.Clamp!);
        _depthGradient = new LinearGradient(
            0,
            Height * 0.45f,
            0,
            Height,
            AColor.Transparent,
            AColor.Argb((int)(255 * Math.Clamp(_shadow * 0.16f, 0, 0.3f)), 0, 0, 0),
            Shader.TileMode.Clamp!);
        _fallbackShadersDirty = false;
    }

    private void ReportTier(string description)
    {
        if (_tierReported) return;
        _tierReported = true;
        global::Android.Util.Log.Info("FlareLiquidGlass", $"Active renderer: {description} (API {(int)Build.VERSION.SdkInt}).");
    }

    private void ReleaseEffects()
    {
        if (OperatingSystem.IsAndroidVersionAtLeast(31)) ClearRenderNodeEffect();
        _renderEffect?.Dispose();
        _renderEffect = null;
        _runtimeShader?.Dispose();
        _runtimeShader = null;
    }

    [SupportedOSPlatform("android31.0")]
    private void ClearRenderNodeEffect() => _renderNode?.SetRenderEffect(null);

    private void ReleaseGraphicsResources()
    {
        ReleaseEffects();
        _renderNode?.Dispose();
        _renderNode = null;
        _backdrop?.Dispose();
        _backdrop = null;
        _baseGradient?.Dispose();
        _baseGradient = null;
        _specularGradient?.Dispose();
        _specularGradient = null;
        _depthGradient?.Dispose();
        _depthGradient = null;
    }

    protected override void Dispose(bool disposing)
    {
        if (disposing)
        {
            ReleaseGraphicsResources();
            _bitmapPaint.Dispose();
            _fillPaint.Dispose();
            _strokePaint.Dispose();
            _clipPath.Dispose();
            _specularMatrix.Dispose();
        }

        base.Dispose(disposing);
    }
}
