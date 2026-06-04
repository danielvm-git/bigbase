/**
 * Design-token type unions.
 *
 * Mirror `react-stubs/tokens.ts` from the prototype so consumers can reference
 * the scale at the type level. The runtime values match the CSS custom property
 * scale in `ui/src/styles/tokens.css`:
 *   --space-0  -> SpaceToken 0
 *   --radius-m -> RadiusToken 'm'
 *   --shadow-l -> ShadowToken 'l'
 *
 * Pure type-level addition. No runtime consumers required — these are kept as
 * frozen arrays for tooling (test assertions, storybook controls, etc.).
 */

export const SPACE_TOKENS = [0, 1, 2, 3, 4, 5, 6, 8, 10, 12, 16, 20, 24] as const
export type SpaceToken = (typeof SPACE_TOKENS)[number]

export const RADIUS_TOKENS = ['xs', 's', 'm', 'l', 'full'] as const
export type RadiusToken = (typeof RADIUS_TOKENS)[number]

export const SHADOW_TOKENS = ['xs', 's', 'm', 'l', 'xl'] as const
export type ShadowToken = (typeof SHADOW_TOKENS)[number]
