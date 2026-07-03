package models

import "time"

type Font struct {
	ID               int64     `json:"id"`
	Filename         string    `json:"filename"`
	DisplayName      string    `json:"display_name"`
	OriginalFilename string    `json:"original_filename"`
	FileHash         string    `json:"file_hash"`
	CreatedAt        time.Time `json:"created_at"`
}
