package models

import "time"

type Control struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Type        string  `json:"type"`
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

type ControlRow struct {
	Control
	TemplateID int64 `json:"-"`
	SortOrder  int   `json:"-"`
}

type RuleRow struct {
	ID         string      `json:"id"`
	TemplateID int64       `json:"-"`
	Type       string      `json:"type"`
	Name       string      `json:"name"`
	Target     string      `json:"target"`
	ConfigJSON string      `json:"-"`
	Config     interface{} `json:"config"`
	SortOrder  int         `json:"-"`
}

type Template struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	BgImage   string    `json:"bg_image"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"created_at"`
}

type TemplateWithHandwriting struct {
	Template
	Controls    []Control          `json:"controls"`
	Rules       []RuleRow          `json:"rules"`
	Handwriting *HandwritingConfig `json:"handwriting,omitempty"`
}

type TemplateListItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
