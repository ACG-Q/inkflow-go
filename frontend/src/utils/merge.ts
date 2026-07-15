import type { HandwritingConfig } from '@/types'

export function mergeHW(base: HandwritingConfig, over: Partial<HandwritingConfig>): HandwritingConfig {
  const r = { ...base }
  for (const [k, v] of Object.entries(over)) {
    if (v !== undefined && v !== null) {
      (r as Record<string, any>)[k] = v
    }
  }
  return r
}
