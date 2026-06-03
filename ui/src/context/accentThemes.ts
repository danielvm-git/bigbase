export type AccentId =
  | 'default'
  | 'january'
  | 'february'
  | 'march'
  | 'april'
  | 'may'
  | 'june'
  | 'july'
  | 'august'
  | 'september'
  | 'october'
  | 'november'
  | 'december'

export interface AccentTheme {
  id: AccentId
  label: string
  brand500: string
  brand600: string
  brand700: string
  rainbow?: boolean
}

export const ACCENT_THEMES: AccentTheme[] = [
  { id: 'default', label: 'Indigo (default)', brand500: '79, 70, 229', brand600: '67, 56, 202', brand700: '55, 48, 163' },
  { id: 'january', label: 'January — Teal', brand500: '13, 148, 136', brand600: '15, 118, 110', brand700: '17, 94, 89' },
  { id: 'february', label: 'February — Orange', brand500: '234, 88, 12', brand600: '194, 65, 12', brand700: '154, 52, 18' },
  { id: 'march', label: 'March — Purple', brand500: '124, 58, 237', brand600: '109, 40, 217', brand700: '91, 33, 182' },
  { id: 'april', label: 'April — Green', brand500: '22, 163, 74', brand600: '21, 128, 61', brand700: '22, 101, 52' },
  { id: 'may', label: 'May — Lavender', brand500: '167, 139, 250', brand600: '139, 92, 246', brand700: '124, 58, 237' },
  { id: 'june', label: 'June — Rainbow', brand500: '79, 70, 229', brand600: '67, 56, 202', brand700: '55, 48, 163', rainbow: true },
  { id: 'july', label: 'July — Peach', brand500: '253, 186, 116', brand600: '251, 146, 60', brand700: '234, 88, 12' },
  { id: 'august', label: 'August — Silver', brand500: '156, 163, 175', brand600: '107, 114, 128', brand700: '75, 85, 99' },
  { id: 'september', label: 'September — Yellow', brand500: '234, 179, 8', brand600: '202, 138, 4', brand700: '161, 98, 7' },
  { id: 'october', label: 'October — Pink', brand500: '236, 72, 153', brand600: '219, 39, 119', brand700: '190, 24, 93' },
  { id: 'november', label: 'November — Blue', brand500: '37, 99, 235', brand600: '29, 78, 216', brand700: '30, 64, 175' },
  { id: 'december', label: 'December — Red', brand500: '220, 38, 38', brand600: '185, 28, 28', brand700: '153, 27, 27' },
]

export const ACCENT_IDS = ACCENT_THEMES.map(t => t.id)

export function isAccentId(value: string): value is AccentId {
  return (ACCENT_IDS as string[]).includes(value)
}

export function getAccentTheme(id: AccentId): AccentTheme {
  return ACCENT_THEMES.find(t => t.id === id) ?? ACCENT_THEMES[0]
}
