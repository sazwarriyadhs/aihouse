import 'package:flutter/material.dart';

class AppColors {
  AppColors._();

  static const background = Color(0xFFF5F7FA);
  static const surface = Color(0xFFFFFFFF);
  static const primary = Color(0xFF1070FF);
  static const navy = Color(0xFF001030);
  static const gold = Color(0xFFE0B060);
  static const text = Color(0xFF102030);
  static const muted = Color(0xFF737783);
  static const border = Color(0xFFEAEBEF);
  static const selected = Color(0xFFEAF2FF);
  static const success = Color(0xFF36B37E);
}

class AppTheme {
  AppTheme._();

  static ThemeData get light {
    const scheme = ColorScheme.light(
      primary: AppColors.primary,
      onPrimary: Colors.white,
      secondary: AppColors.gold,
      onSecondary: AppColors.navy,
      surface: AppColors.surface,
      onSurface: AppColors.text,
      error: Color(0xFFD92D20),
      onError: Colors.white,
    );

    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,
      colorScheme: scheme,
      scaffoldBackgroundColor: AppColors.background,
      fontFamily: 'Roboto',
      appBarTheme: const AppBarTheme(
        backgroundColor: AppColors.surface,
        foregroundColor: AppColors.text,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        centerTitle: false,
      ),
      cardColor: AppColors.surface,
      dividerColor: AppColors.border,
      iconTheme: const IconThemeData(color: AppColors.muted),
      textTheme: const TextTheme(
        headlineSmall: TextStyle(
          color: AppColors.text,
          fontSize: 23,
          fontWeight: FontWeight.w700,
        ),
        titleLarge: TextStyle(
          color: AppColors.text,
          fontSize: 18,
          fontWeight: FontWeight.w700,
        ),
        titleMedium: TextStyle(
          color: AppColors.text,
          fontSize: 14,
          fontWeight: FontWeight.w700,
        ),
        bodyMedium: TextStyle(color: AppColors.text, fontSize: 13),
        bodySmall: TextStyle(color: AppColors.muted, fontSize: 11),
      ),
    );
  }
}
