package com.cornytuna.soundboard.core.ui.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.em
import androidx.compose.ui.unit.sp

// Generado por brand/scripts/build.mjs desde tokens.json. No editar a mano.
// Display: Bricolage Grotesque. Agrégala en res/font y reemplaza FontFamily.Default.
val DisplayFont: FontFamily = FontFamily.Default

val Typography = Typography(
    displayLarge = TextStyle(
        fontFamily = DisplayFont,
        fontWeight = FontWeight.ExtraBold,
        fontSize = 56.sp,
        lineHeight = 1.02.em,
        letterSpacing = -0.03.em
    ),
    titleLarge = TextStyle(
        fontFamily = DisplayFont,
        fontWeight = FontWeight.Bold,
        fontSize = 22.sp,
        lineHeight = 1.273.em,
        letterSpacing = -0.01.em
    ),
    titleMedium = TextStyle(
        fontFamily = DisplayFont,
        fontWeight = FontWeight.Bold,
        fontSize = 20.sp,
        lineHeight = 1.28.em,
        letterSpacing = -0.01.em
    ),
    bodyLarge = TextStyle(
        fontFamily = FontFamily.Default,
        fontWeight = FontWeight.Normal,
        fontSize = 16.sp,
        lineHeight = 1.5.em,
        letterSpacing = 0.em
    ),
    labelMedium = TextStyle(
        fontFamily = FontFamily.Default,
        fontWeight = FontWeight.SemiBold,
        fontSize = 12.sp,
        lineHeight = 1.333.em,
        letterSpacing = 0.015.em
    )
)
