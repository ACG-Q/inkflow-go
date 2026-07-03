package core

import (
	"image"
	"image/draw"
	"math/rand"
)

type EffectParams struct {
	Noise  int `json:"noise"`
	Yellow int `json:"yellow"`
	Crease int `json:"crease"`
}

type Preset struct {
	Name   string
	Params EffectParams
}

var Presets = []Preset{
	{"none", EffectParams{0, 0, 0}},
	{"light", EffectParams{15, 20, 10}},
	{"moderate", EffectParams{35, 40, 30}},
	{"heavy", EffectParams{60, 65, 55}},
}

func ResolvePreset(name string) EffectParams {
	for _, p := range Presets {
		if p.Name == name {
			return p.Params
		}
	}
	return Presets[1].Params
}

func Compose(bg, textLayer *image.RGBA, p EffectParams, seed int64) *image.RGBA {
	b := bg.Bounds()
	out := image.NewRGBA(b)
	draw.Draw(out, b, bg, b.Min, draw.Src)
	draw.Draw(out, b, textLayer, b.Min, draw.Over)
	if p.Noise > 0 || p.Yellow > 0 || p.Crease > 0 {
		rng := rand.New(rand.NewSource(seed))
		applyNoise(out, p.Noise, rng)
		applyYellow(out, p.Yellow)
		applyCrease(out, p.Crease, rng)
	}
	return out
}

func applyNoise(img *image.RGBA, intensity int, rng *rand.Rand) {
	if intensity == 0 {
		return
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if rng.Intn(100) < intensity/5 {
				off := img.PixOffset(x, y)
				v := uint8(rng.Intn(40))
				img.Pix[off+0] = clamp(img.Pix[off+0] + v)
				img.Pix[off+1] = clamp(img.Pix[off+1] + v)
				img.Pix[off+2] = clamp(img.Pix[off+2] + v)
			}
		}
	}
}

func applyYellow(img *image.RGBA, intensity int) {
	if intensity == 0 {
		return
	}
	f := float64(intensity) / 100
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			off := img.PixOffset(x, y)
			img.Pix[off+0] = clamp(img.Pix[off+0] - uint8(5*f))
			img.Pix[off+2] = clamp(img.Pix[off+2] + uint8(10*f))
		}
	}
}

func applyCrease(img *image.RGBA, intensity int, rng *rand.Rand) {
	if intensity == 0 {
		return
	}
}

func clamp(v uint8) uint8 {
	if int(v) > 255 {
		return 255
	}
	return v
}
