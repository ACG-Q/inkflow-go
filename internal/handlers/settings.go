package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"inkflow-go/internal/models"
)

func GetHandwritingSettings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var value string
		err := db.QueryRow("SELECT value FROM settings WHERE key = 'handwriting'").Scan(&value)
		if err == sql.ErrNoRows {
			defaults := models.DefaultHandwriting()
			Success(c, defaults)
			return
		}
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		var hw models.HandwritingConfig
		if err := json.Unmarshal([]byte(value), &hw); err != nil {
			Error(c, http.StatusInternalServerError, 50002, "配置数据格式错误")
			return
		}
		Success(c, hw)
	}
}

func UpdateHandwritingSettings(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var hw models.HandwritingConfig
		if err := c.ShouldBindJSON(&hw); err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		b, err := json.Marshal(hw)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50002, "配置序列化失败")
			return
		}
		_, err = db.Exec("INSERT INTO settings (key, value) VALUES ('handwriting', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", string(b))
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Success(c, nil)
	}
}

const defaultFontBatchLimit = 10

// readFontBatchLimit returns the batch font import limit from settings.
// A value of 0 means unlimited. Falls back to defaultFontBatchLimit.
func readFontBatchLimit(db *sql.DB) int {
	var value string
	if err := db.QueryRow("SELECT value FROM settings WHERE key = 'font_batch_limit'").Scan(&value); err != nil {
		return defaultFontBatchLimit
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return defaultFontBatchLimit
	}
	if n < 0 {
		return 0
	}
	return n
}

func GetFontBatchLimit(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		Success(c, gin.H{"limit": readFontBatchLimit(db)})
	}
}

func UpdateFontBatchLimit(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Limit int `json:"limit"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Limit < 0 {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		_, err := db.Exec("INSERT INTO settings (key, value) VALUES ('font_batch_limit', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", strconv.Itoa(req.Limit))
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Success(c, nil)
	}
}
