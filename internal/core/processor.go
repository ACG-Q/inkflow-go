package core

import (
	"image"
	"image/draw"
)

func Compose(bg, textLayer *image.RGBA, seed int64) *image.RGBA {
	b := bg.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, bg, b.Min, draw.Src)
	draw.Draw(out, b, textLayer, b.Min, draw.Over)
	return out
}
