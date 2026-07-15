import type { Control, HandwritingConfig } from '@/types'

export function drawPaperTexture(ctx: CanvasRenderingContext2D, w: number, h: number, hw: HandwritingConfig) {
  if (hw.paper_enabled === false) return
  ctx.save()
  ctx.globalCompositeOperation = 'source-over'
  const opacity = hw.paper_opacity ?? 0.12
  ctx.fillStyle = `rgba(248, 245, 230, ${opacity})`
  ctx.fillRect(0, 0, w, h)
  const fibers = hw.fiber_count ?? 200
  for (let i = 0; i < fibers; i++) {
    ctx.beginPath()
    ctx.moveTo(Math.random() * w, Math.random() * h)
    ctx.lineTo(Math.random() * w, Math.random() * h)
    ctx.strokeStyle = `rgba(100, 80, 50, ${Math.random() * 0.08})`
    ctx.lineWidth = 0.5 + Math.random() * 1
    ctx.stroke()
  }
  const dots = hw.dot_count ?? 800
  for (let i = 0; i < dots; i++) {
    ctx.fillStyle = `rgba(80, 60, 40, ${Math.random() * 0.1})`
    ctx.fillRect(Math.random() * w, Math.random() * h, 1, 1)
  }
  ctx.restore()
}

export function drawTextOnContext(ctx: CanvasRenderingContext2D, text: string, ctrl: Control, hw: HandwritingConfig) {
  ctx.save()
  const fontSize = ctrl.fontSize || 28
  let fontFamily = ctrl.fontFamily || 'sans-serif'
  if (fontFamily === 'sans-serif' || fontFamily === 'serif' || fontFamily === 'monospace') {
    fontFamily = `${fontFamily}, sans-serif`
  } else {
    fontFamily = `'${fontFamily}', sans-serif`
  }
  ctx.font = `${fontSize}px ${fontFamily}`
  ctx.textBaseline = 'top'

  const globalTilt = (Math.random() - 0.5) * 2 * (hw.global_tilt ?? 1)
  ctx.translate(ctrl.x, ctrl.y)
  ctx.rotate(globalTilt * Math.PI / 180)

  const baselineDrift = hw.baseline_drift ?? 0.8
  const charJitter = hw.char_jitter ?? 2
  const charRotation = hw.char_rotation ?? 1.5
  const inkMin = hw.ink_opacity_min ?? 0.85
  const inkRange = (hw.ink_opacity_max ?? 1) - inkMin
  const charSpacing = hw.char_spacing ?? 1.5
  const shadowBlur = hw.shadow_blur ?? 0.8
  const inkSpotsEnabled = hw.ink_spots_enabled !== false
  const inkSpotsChance = hw.ink_spots_chance ?? 0.15
  const inkSpotsMax = hw.ink_spots_max ?? 2

  let xOffset = 0
  let baseLineY = 0
  for (const char of text.split('')) {
    if (char === ' ') { xOffset += fontSize * 0.4; continue }
    baseLineY += (Math.random() - 0.5) * 2 * baselineDrift
    baseLineY = Math.min(Math.max(baseLineY, -2), 2)
    const jitterY = (Math.random() - 0.5) * charJitter + baseLineY
    const cRot = (Math.random() - 0.5) * charRotation * 2 * Math.PI / 180
    const inkOpacity = inkMin + Math.random() * inkRange

    ctx.save()
    ctx.translate(xOffset, jitterY)
    ctx.rotate(cRot)
    ctx.shadowBlur = shadowBlur
    ctx.shadowColor = 'rgba(0,0,0,0.15)'
    ctx.fillStyle = `rgba(20, 20, 25, ${inkOpacity})`
    ctx.fillText(char, 0, 0)
    ctx.shadowBlur = 0
    ctx.restore()

    if (inkSpotsEnabled && Math.random() < inkSpotsChance) {
      const spotCount = Math.floor(Math.random() * inkSpotsMax) + 1
      for (let s = 0; s < spotCount; s++) {
        ctx.save()
        ctx.translate(xOffset + (Math.random() - 0.5) * fontSize * 0.4, jitterY + (Math.random() - 0.5) * fontSize * 0.3)
        ctx.beginPath()
        ctx.arc(0, 0, Math.random() * 1.5 + 0.5, 0, Math.PI * 2)
        ctx.fillStyle = `rgba(30, 30, 35, ${0.3 + Math.random() * 0.4})`
        ctx.fill()
        ctx.restore()
      }
    }
    xOffset += ctx.measureText(char).width + (Math.random() - 0.5) * charSpacing
  }
  ctx.restore()
}

export function drawCheckOnContext(ctx: CanvasRenderingContext2D, ctrl: Control, hw: HandwritingConfig) {
  if (hw.checkbox_enabled === false) {
    const size = ctrl.checkSize || ctrl.width || 32
    ctx.fillStyle = '#000'
    ctx.fillRect(ctrl.x, ctrl.y, size, size)
    return
  }
  ctx.save()
  const size = ctrl.checkSize || ctrl.width || 32
  const cx = ctrl.x + size / 2, cy = ctrl.y + size / 2
  const o = size * 0.25
  const r = () => (Math.random() - 0.5) * size * 0.03
  ctx.beginPath()
  ctx.moveTo(cx - o + r(), cy + r())
  ctx.quadraticCurveTo(cx - o + 3 + r(), cy + 4 + r(), cx - o * 0.4 + r(), cy + o * 0.9 + r())
  ctx.quadraticCurveTo(cx - o * 0.4 + 2 + r(), cy + o * 0.9 - 2 + r(), cx + o + r(), cy - o * 0.6 + r())
  ctx.lineWidth = Math.max(3, size * 0.12)
  ctx.strokeStyle = '#1a1a1a'
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.shadowBlur = 1.2
  ctx.shadowColor = 'rgba(0,0,0,0.3)'
  ctx.stroke()
  ctx.shadowBlur = 0
  ctx.restore()
}

export function renderPreview(
  canvas: HTMLCanvasElement,
  bgImg: HTMLImageElement | null,
  controls: Control[],
  formData: Record<string, any>,
  hiddenFields: Set<string>,
  hw: HandwritingConfig,
) {
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  if (bgImg && bgImg.complete && bgImg.naturalWidth > 0) {
    ctx.drawImage(bgImg, 0, 0, canvas.width, canvas.height)
  } else {
    ctx.fillStyle = '#fff'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
  }
  drawPaperTexture(ctx, canvas.width, canvas.height, hw)
  for (const ctrl of controls) {
    if (hiddenFields.has(ctrl.id)) continue
    if (ctrl.type === 'textbox') {
      const val = formData[ctrl.id]
      if (val && String(val).trim()) drawTextOnContext(ctx, String(val).trim(), ctrl, hw)
    } else if (ctrl.type === 'checkbox') {
      if (formData[ctrl.id]) drawCheckOnContext(ctx, ctrl, hw)
    }
  }
}

export function generateTextLayerDataURL(
  width: number,
  height: number,
  controls: Control[],
  formData: Record<string, any>,
  hiddenFields: Set<string>,
  hw: HandwritingConfig,
): string {
  const offCanvas = document.createElement('canvas')
  offCanvas.width = width
  offCanvas.height = height
  const offCtx = offCanvas.getContext('2d')
  if (!offCtx) return ''
  offCtx.clearRect(0, 0, offCanvas.width, offCanvas.height)
  drawPaperTexture(offCtx, offCanvas.width, offCanvas.height, hw)
  for (const ctrl of controls) {
    if (hiddenFields.has(ctrl.id)) continue
    if (ctrl.type === 'textbox') {
      const val = formData[ctrl.id]
      if (val && String(val).trim()) drawTextOnContext(offCtx, String(val).trim(), ctrl, hw)
    } else if (ctrl.type === 'checkbox') {
      if (formData[ctrl.id]) drawCheckOnContext(offCtx, ctrl, hw)
    }
  }
  return offCanvas.toDataURL('image/png')
}
