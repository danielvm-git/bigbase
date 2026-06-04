import { describe, it, expect, expectTypeOf } from 'vitest'
import {
  SPACE_TOKENS,
  RADIUS_TOKENS,
  SHADOW_TOKENS,
  type SpaceToken,
  type RadiusToken,
  type ShadowToken,
} from './tokens'

describe('design token unions', () => {
  describe('SpaceToken', () => {
    it('contains all numeric scale values from tokens.css', () => {
      // Mirrors --space-0..--space-24 in ui/src/styles/tokens.css
      expect(SPACE_TOKENS).toEqual([0, 1, 2, 3, 4, 5, 6, 8, 10, 12, 16, 20, 24])
    })

    it('narrows a literal to the SpaceToken type', () => {
      const x: SpaceToken = 6
      expect(x).toBe(6)
      expectTypeOf<SpaceToken>().toEqualTypeOf<
        0 | 1 | 2 | 3 | 4 | 5 | 6 | 8 | 10 | 12 | 16 | 20 | 24
      >()
    })
  })

  describe('RadiusToken', () => {
    it('contains all radius scale values from tokens.css', () => {
      // Mirrors --radius-xs..--radius-full in ui/src/styles/tokens.css
      expect(RADIUS_TOKENS).toEqual(['xs', 's', 'm', 'l', 'full'])
    })

    it('narrows a literal to the RadiusToken type', () => {
      const x: RadiusToken = 'm'
      expect(x).toBe('m')
      expectTypeOf<RadiusToken>().toEqualTypeOf<'xs' | 's' | 'm' | 'l' | 'full'>()
    })
  })

  describe('ShadowToken', () => {
    it('contains all shadow scale values from tokens.css', () => {
      // Mirrors --shadow-xs..--shadow-xl in ui/src/styles/tokens.css
      expect(SHADOW_TOKENS).toEqual(['xs', 's', 'm', 'l', 'xl'])
    })

    it('narrows a literal to the ShadowToken type', () => {
      const x: ShadowToken = 'l'
      expect(x).toBe('l')
      expectTypeOf<ShadowToken>().toEqualTypeOf<'xs' | 's' | 'm' | 'l' | 'xl'>()
    })
  })
})
