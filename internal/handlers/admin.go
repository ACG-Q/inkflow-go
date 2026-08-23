package handlers

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func StatsHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var templateCount, recordCount, fontCount int64
		db.QueryRow("SELECT COUNT(*) FROM templates WHERE deleted_at IS NULL").Scan(&templateCount)
		db.QueryRow("SELECT COUNT(*) FROM signing_records WHERE deleted_at IS NULL").Scan(&recordCount)
		db.QueryRow("SELECT COUNT(*) FROM fonts WHERE deleted_at IS NULL").Scan(&fontCount)
		Success(c, gin.H{
			"template_count": templateCount,
			"record_count":   recordCount,
			"font_count":     fontCount,
		})
	}
}
