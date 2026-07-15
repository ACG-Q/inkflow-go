export type ControlType = 'textbox' | 'checkbox'

export interface Control {
  id: string
  label: string
  type: ControlType
  x: number
  y: number
  width: number
  height: number
  fontSize: number
  fontFamily: string
  handwritingFont?: string
  required: boolean
  previewText?: string
  checkSize?: number
}

export interface HandwritingConfig {
  font_family?: string
  paper_enabled?: boolean
  paper_opacity?: number
  fiber_count?: number
  dot_count?: number
  global_tilt?: number
  baseline_drift?: number
  char_jitter?: number
  char_rotation?: number
  ink_opacity_min?: number
  ink_opacity_max?: number
  char_spacing?: number
  ink_spots_enabled?: boolean
  ink_spots_chance?: number
  ink_spots_max?: number
  shadow_blur?: number
  checkbox_enabled?: boolean
}

export type RuleConfig =
  | { targets?: string[] }
  | { groups?: string[][] }
  | { source?: string; condition?: string; value?: any; action?: string }
  | { source?: string; direction?: string; transform?: string }
  | { source?: string; operation?: string; operand?: number }
  | { source?: string; format?: string }
  | { default_value?: any }
  | { validation_type?: string; pattern?: string; min_length?: number; max_length?: number }
  | Record<string, any>

export interface Rule {
  id: string
  type: string
  name?: string
  target: string
  config: RuleConfig
}

export interface Template {
  id: number
  name: string
  bg_image: string
  width: number
  height: number
  controls: Control[]
  rules?: Rule[]
  handwriting?: HandwritingConfig
  created_at: string
}

export interface TemplateListItem {
  id: number
  name: string
  bg_image?: string
  created_at: string
}

export interface SigningRecord {
  id: number
  template_id: number
  template_name: string
  fields_data: string
  image_url: string
  ip: string
  created_at: string
}

export interface FontItem {
  id: number
  filename: string
  display_name: string
  original_filename: string
  file_hash?: string
  created_at: string
}

export interface AdminStats {
  template_count: number
  record_count: number
  font_count: number
}

export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

export interface PageData<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}
