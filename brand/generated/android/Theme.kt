package com.cornytuna.soundboard.core.ui.theme

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp

// Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano.
private val LightColorScheme = lightColorScheme(
    primary = BrandPrimaryLight,
    onPrimary = BrandOnPrimaryLight,
    secondary = BrandSecondaryLight,
    onSecondary = BrandOnSecondaryLight,
    tertiary = BrandTertiaryLight,
    onTertiary = BrandOnTertiaryLight,
    background = BrandBackgroundLight,
    onBackground = BrandOnBackgroundLight,
    surface = BrandSurfaceLight,
    onSurface = BrandOnSurfaceLight,
    surfaceVariant = BrandSurfaceVariantLight,
    onSurfaceVariant = BrandOnSurfaceVariantLight,
    outline = BrandOutlineLight,
    error = BrandErrorLight
)

private val DarkColorScheme = darkColorScheme(
    primary = BrandPrimaryDark,
    onPrimary = BrandOnPrimaryDark,
    secondary = BrandSecondaryDark,
    onSecondary = BrandOnSecondaryDark,
    tertiary = BrandTertiaryDark,
    onTertiary = BrandOnTertiaryDark,
    background = BrandBackgroundDark,
    onBackground = BrandOnBackgroundDark,
    surface = BrandSurfaceDark,
    onSurface = BrandOnSurfaceDark,
    surfaceVariant = BrandSurfaceVariantDark,
    onSurfaceVariant = BrandOnSurfaceVariantDark,
    outline = BrandOutlineDark,
    error = BrandErrorDark
)

private val BrandShapes = Shapes(
    small = RoundedCornerShape(6.dp),
    medium = RoundedCornerShape(8.dp),
    large = RoundedCornerShape(14.dp)
)

// El claro es el de entrada; el oscuro lo elige la persona (no sigue al sistema).
@Composable
fun CastBerryTheme(
    darkTheme: Boolean = false,
    content: @Composable () -> Unit
) {
    val colorScheme = if (darkTheme) DarkColorScheme else LightColorScheme

    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        shapes = BrandShapes,
        content = content
    )
}
