import type { HandwritingConfig } from '@/types'

export const HW_DEFAULTS: HandwritingConfig = {
  font_family: 'sans-serif',
  paper_enabled: true, paper_opacity: 0.12, fiber_count: 200, dot_count: 800,
  global_tilt: 1, baseline_drift: 0.8, char_jitter: 2, char_rotation: 1.5,
  ink_opacity_min: 0.85, ink_opacity_max: 1.0, char_spacing: 1.5,
  ink_spots_enabled: true, ink_spots_chance: 0.15, ink_spots_max: 2,
  shadow_blur: 0.8, checkbox_enabled: true,
}

export const ALLOWED_FONT_EXTS = ['.ttf', '.otf', '.woff', '.woff2'] as const
