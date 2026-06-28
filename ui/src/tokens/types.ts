export interface ColorTokens {
  neutral: Record<string, string>
  brand: Record<string, string>
  success: string
  successBg: string
  successFg: string
  warning: string
  warningBg: string
  warningFg: string
  error: string
  errorBg: string
  errorFg: string
  info: string
  infoBg: string
  infoFg: string
}

export interface BackgroundTokens {
  default: string
  surface: string
  surfaceHover: string
  surfaceSecondary: string
  accent: string
  accentHover: string
  accentActive: string
  subtle: string
}

export interface ForegroundTokens {
  primary: string
  secondary: string
  tertiary: string
  accent: string
  onAccent: string
}

export interface BorderTokens {
  default: string
  strong: string
  focus: string
  accent: string
  error: string
}

export interface SpacingTokens {
  0: string
  1: string
  2: string
  3: string
  4: string
  5: string
  6: string
  8: string
  10: string
  12: string
  16: string
  20: string
  24: string
}

export interface RadiusTokens {
  xs: string
  s: string
  m: string
  l: string
  full: string
}

export interface TypographyTokens {
  fontSans: string
  fontMono: string
  textXs: string
  textS: string
  textM: string
  textL: string
  textXl: string
  text2xl: string
  text3xl: string
}

export interface ShadowTokens {
  xs: string
  s: string
  m: string
  l: string
  xl: string
}

export interface MotionTokens {
  durationFast: string
  durationShort: string
  durationMedium: string
  durationExtended: string
  durationSlow: string
  easeStandard: string
  easeEmphasized: string
  easeInOut: string
  easeOut: string
}

export interface DesignTokens {
  colors: ColorTokens
  bg: BackgroundTokens
  fg: ForegroundTokens
  border: BorderTokens
  spacing: SpacingTokens
  radius: RadiusTokens
  typography: TypographyTokens
  shadow: ShadowTokens
  motion: MotionTokens
  focusRing: string
}
