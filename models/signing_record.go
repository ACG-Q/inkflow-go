package models

import "time"

type SigningRecord struct {
	ID           int64     `json:"id"`
	TemplateID   int64     `json:"template_id"`
	TemplateName string    `json:"template_name,omitempty"`
	FieldsData   string    `json:"fields_data"`
	ImageURL     string    `json:"image_url"`
	IP           string    `json:"ip"`
	CreatedAt    time.Time `json:"created_at"`
}
