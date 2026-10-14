package com.cornytuna.soundboard.core.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp

// Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano.
private val DarkColorScheme = darkColorScheme(
    primary = DeckPrimaryDark,
    onPrimary = DeckOnPrimaryDark,
    secondary = DeckSecondaryDark,
    onSecondary = DeckOnSecondaryDark,
    tertiary = DeckTertiaryDark,
    onTertiary = DeckOnTertiaryDark,
    background = DeckBackgroundDark,
    onBackground = DeckOnBackgroundDark,
    surface = DeckSurfaceDark,
    onSurface = DeckOnSurfaceDark,
    surfaceVariant = DeckSurfaceVariantDark,
    onSurfaceVariant = DeckOnSurfaceVariantDark,
    outline = DeckOutlineDark,
    error = DeckErrorDark
)

private val LightColorScheme = lightColorScheme(
    primary = DeckPrimaryLight,
    onPrimary = DeckOnPrimaryLight,
    secondary = DeckSecondaryLight,
    onSecondary = DeckOnSecondaryLight,
    tertiary = DeckTertiaryLight,
    onTertiary = DeckOnTertiaryLight,
    background = DeckBackgroundLight,
    onBackground = DeckOnBackgroundLight,
    surface = DeckSurfaceLight,
    onSurface = DeckOnSurfaceLight,
    surfaceVariant = DeckSurfaceVariantLight,
    onSurfaceVariant = DeckOnSurfaceVariantLight,
    outline = DeckOutlineLight,
    error = DeckErrorLight
)

private val DeckShapes = Shapes(
    small = RoundedCornerShape(10.dp),
    medium = RoundedCornerShape(16.dp),
    large = RoundedCornerShape(22.dp)
)

@Composable
fun SoundboardTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit
) {
    val colorScheme = if (darkTheme) DarkColorScheme else LightColorScheme

    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        shapes = DeckShapes,
        content = content
    )
}
