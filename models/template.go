package models

import "time"

type Control struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Type        string  `json:"type"` // textbox, checkbox
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	FontSize    int     `json:"font_size"`
	FontFamily  string  `json:"font_family"`
	Required    bool    `json:"required"`
	PreviewText string  `json:"preview_text"`
	CheckSize   int     `json:"check_size"`
}

type Rule struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"` // required, validation, mutual_exclusion, trigger, value_sync, arithmetic, auto_fill
	ControlID string      `json:"control_id"`
	Config    interface{} `json:"config"`
}

type TemplateConfig struct {
	Controls []Control `json:"controls"`
	Rules    []Rule    `json:"rules"`
}

type Template struct {
	ID         int64          `json:"id"`
	Name       string         `json:"name"`
	BgImage    string         `json:"bg_image"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	ConfigJSON string         `json:"config_json"`
	CreatedAt  time.Time      `json:"created_at"`
}

type TemplateListItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
