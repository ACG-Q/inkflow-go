package models

type HandwritingConfig struct {
	FontFamily       *string  `json:"font_family,omitempty"`
	PaperEnabled     *bool    `json:"paper_enabled,omitempty"`
	PaperOpacity     *float64 `json:"paper_opacity,omitempty"`
	FiberCount       *int     `json:"fiber_count,omitempty"`
	DotCount         *int     `json:"dot_count,omitempty"`
	GlobalTilt       *float64 `json:"global_tilt,omitempty"`
	BaselineDrift    *float64 `json:"baseline_drift,omitempty"`
	CharJitter       *float64 `json:"char_jitter,omitempty"`
	CharRotation     *float64 `json:"char_rotation,omitempty"`
	InkOpacityMin    *float64 `json:"ink_opacity_min,omitempty"`
	InkOpacityMax    *float64 `json:"ink_opacity_max,omitempty"`
	CharSpacing      *float64 `json:"char_spacing,omitempty"`
	InkSpotsEnabled  *bool    `json:"ink_spots_enabled,omitempty"`
	InkSpotsChance   *float64 `json:"ink_spots_chance,omitempty"`
	InkSpotsMax      *int     `json:"ink_spots_max,omitempty"`
	ShadowBlur       *float64 `json:"shadow_blur,omitempty"`
	CheckboxEnabled  *bool    `json:"checkbox_enabled,omitempty"`
}

func DefaultHandwriting() HandwritingConfig {
	t := true
	return HandwritingConfig{
		FontFamily:       SP("sans-serif"),
		PaperEnabled:     &t,
		PaperOpacity:     FP(0.12),
		FiberCount:       IP(200),
		DotCount:         IP(800),
		GlobalTilt:       FP(1),
		BaselineDrift:    FP(0.8),
		CharJitter:       FP(2),
		CharRotation:     FP(1.5),
		InkOpacityMin:    FP(0.85),
		InkOpacityMax:    FP(1.0),
		CharSpacing:      FP(1.5),
		InkSpotsEnabled:  &t,
		InkSpotsChance:   FP(0.15),
		InkSpotsMax:      IP(2),
		ShadowBlur:       FP(0.8),
		CheckboxEnabled:  &t,
	}
}

func (base HandwritingConfig) Merge(other HandwritingConfig) HandwritingConfig {
	r := base
	if other.FontFamily != nil {
		r.FontFamily = other.FontFamily
	}
	if other.PaperEnabled != nil {
		r.PaperEnabled = other.PaperEnabled
	}
	if other.PaperOpacity != nil {
		r.PaperOpacity = other.PaperOpacity
	}
	if other.FiberCount != nil {
		r.FiberCount = other.FiberCount
	}
	if other.DotCount != nil {
		r.DotCount = other.DotCount
	}
	if other.GlobalTilt != nil {
		r.GlobalTilt = other.GlobalTilt
	}
	if other.BaselineDrift != nil {
		r.BaselineDrift = other.BaselineDrift
	}
	if other.CharJitter != nil {
		r.CharJitter = other.CharJitter
	}
	if other.CharRotation != nil {
		r.CharRotation = other.CharRotation
	}
	if other.InkOpacityMin != nil {
		r.InkOpacityMin = other.InkOpacityMin
	}
	if other.InkOpacityMax != nil {
		r.InkOpacityMax = other.InkOpacityMax
	}
	if other.CharSpacing != nil {
		r.CharSpacing = other.CharSpacing
	}
	if other.InkSpotsEnabled != nil {
		r.InkSpotsEnabled = other.InkSpotsEnabled
	}
	if other.InkSpotsChance != nil {
		r.InkSpotsChance = other.InkSpotsChance
	}
	if other.InkSpotsMax != nil {
		r.InkSpotsMax = other.InkSpotsMax
	}
	if other.ShadowBlur != nil {
		r.ShadowBlur = other.ShadowBlur
	}
	if other.CheckboxEnabled != nil {
		r.CheckboxEnabled = other.CheckboxEnabled
	}
	return r
}

func FP(v float64) *float64 { return &v }
func IP(v int) *int         { return &v }
func SP(v string) *string   { return &v }
