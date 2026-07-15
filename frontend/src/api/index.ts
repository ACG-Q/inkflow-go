import type { ApiResponse, PageData, TemplateListItem, Template, SigningRecord, FontItem, AdminStats, HandwritingConfig } from '@/types'

const BASE = '/api'

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

function clearAuth() {
  localStorage.removeItem('token')
  window.location.href = '/admin/login'
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`

  const isFormData = options?.body instanceof FormData
  if (!isFormData) {
    headers['Content-Type'] = 'application/json'
  }

  const userHeaders = options?.headers as Record<string, string> | undefined
  Object.assign(headers, userHeaders)

  let res: Response
  try {
    res = await fetch(`${BASE}${url}`, {
      ...options,
      headers,
      signal: AbortSignal.timeout(30000),
    })
  } catch {
    throw new ApiError(0, '网络连接失败，请检查网络后重试')
  }

  if (res.status === 401) {
    clearAuth()
    throw new ApiError(401, '登录已过期，请重新登录')
  }

  if (!res.ok) {
    const text = await res.text().catch(() => '未知错误')
    throw new ApiError(res.status, `请求失败: ${res.status} ${text.slice(0, 200)}`)
  }

  return res.json()
}

export const api = {
  login: (data: { username: string; password: string }) =>
    request<ApiResponse<{ token: string }>>('/auth/login', {
      method: 'POST', body: JSON.stringify(data),
    }),
  me: () => request<ApiResponse<{ username: string }>>('/auth/me'),

  listTemplates: (params?: Record<string, string | number>) =>
    request<ApiResponse<PageData<TemplateListItem>>>(`/templates${params ? '?' + new URLSearchParams(
      Object.entries(params).map(([k, v]) => [k, String(v)])
    ).toString() : ''}`),
  createTemplate: (name: string) =>
    request<ApiResponse<{ id: number }>>('/templates', {
      method: 'POST', body: JSON.stringify({ name }),
    }),
  getTemplate: (id: number) => request<ApiResponse<Template>>(`/templates/${id}`),
  updateTemplate: (id: number, data: Partial<Pick<Template, 'name' | 'controls' | 'rules' | 'handwriting' | 'width' | 'height'>>) =>
    request<ApiResponse<null>>(`/templates/${id}`, {
      method: 'PUT', body: JSON.stringify(data),
    }),
  deleteTemplate: (id: number) =>
    request<ApiResponse<null>>(`/templates/${id}`, { method: 'DELETE' }),
  uploadTemplateFile: (id: number, file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<ApiResponse<{ bg_image: string }>>(`/templates/${id}/upload`, {
      method: 'POST', body: fd,
    })
  },

  signTemplate: (id: number, data: {
    fields_data: Record<string, any>
    effect_preset: string
    text_layer_data: string
    handwriting?: HandwritingConfig
  }) =>
    request<ApiResponse<{ image_url: string }>>(`/templates/${id}/sign`, {
      method: 'POST', body: JSON.stringify(data),
    }),

  listRecords: (params?: Record<string, string | number>) =>
    request<ApiResponse<PageData<SigningRecord>>>(`/records${params ? '?' + new URLSearchParams(
      Object.entries(params).map(([k, v]) => [k, String(v)])
    ).toString() : ''}`),
  deleteRecord: (id: number) =>
    request<ApiResponse<null>>(`/records/${id}`, { method: 'DELETE' }),

  listFonts: () => request<ApiResponse<FontItem[]>>('/fonts'),
  uploadFont: (file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<ApiResponse<FontItem>>('/fonts', {
      method: 'POST', body: fd,
    })
  },
  updateFont: (id: number, data: { display_name: string }) =>
    request<ApiResponse<null>>(`/fonts/${id}`, {
      method: 'PUT', body: JSON.stringify(data),
    }),
  deleteFont: (id: number) =>
    request<ApiResponse<null>>(`/fonts/${id}`, { method: 'DELETE' }),

  importTemplate: (file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<ApiResponse<{ id: number; name: string }>>('/templates/import', {
      method: 'POST', body: fd,
    })
  },

  getStats: () => request<ApiResponse<AdminStats>>('/admin/stats'),

  getHandwritingSettings: () =>
    request<ApiResponse<HandwritingConfig>>('/settings/handwriting'),
  updateHandwritingSettings: (data: HandwritingConfig) =>
    request<ApiResponse<null>>('/settings/handwriting', {
      method: 'PUT', body: JSON.stringify(data),
    }),
}
