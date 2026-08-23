package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"inkflow-go/internal/models"
)

func ListRecordsHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		templateID := c.Query("template_id")
		start := c.Query("start")
		end := c.Query("end")
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
		offset := (page - 1) * pageSize

		var total int64
		countSQL := "SELECT COUNT(*) FROM signing_records WHERE deleted_at IS NULL"
		args := []interface{}{}
		if templateID != "" {
			countSQL += " AND template_id = ?"
			args = append(args, templateID)
		}
		if start != "" {
			countSQL += " AND created_at >= ?"
			args = append(args, start)
		}
		if end != "" {
			countSQL += " AND created_at <= ?"
			args = append(args, end)
		}
		db.QueryRow(countSQL, args...).Scan(&total)

		dataSQL := `SELECT sr.id, sr.template_id, COALESCE(t.name,''), COALESCE(sr.fields_data,'{}'), COALESCE(sr.image_url,''), COALESCE(sr.ip_address,''), sr.created_at 
			FROM signing_records sr LEFT JOIN templates t ON sr.template_id = t.id WHERE sr.deleted_at IS NULL`
		if templateID != "" {
			dataSQL += " AND sr.template_id = ?"
		}
		if start != "" {
			dataSQL += " AND sr.created_at >= ?"
		}
		if end != "" {
			dataSQL += " AND sr.created_at <= ?"
		}
		dataSQL += " ORDER BY sr.created_at DESC LIMIT ? OFFSET ?"
		queryArgs := make([]interface{}, len(args))
		copy(queryArgs, args)
		queryArgs = append(queryArgs, pageSize, offset)

		rows, err := db.Query(dataSQL, queryArgs...)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		defer rows.Close()

		items := []models.SigningRecord{}
		for rows.Next() {
			var item models.SigningRecord
			rows.Scan(&item.ID, &item.TemplateID, &item.TemplateName, &item.FieldsData, &item.ImageURL, &item.IP, &item.CreatedAt)
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Page(c, items, total, page, pageSize)
	}
}

func GetRecordHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var item models.SigningRecord
		err = db.QueryRow(`SELECT sr.id, sr.template_id, COALESCE(t.name,''), COALESCE(sr.fields_data,'{}'), COALESCE(sr.image_url,''), COALESCE(sr.ip_address,''), sr.created_at 
			FROM signing_records sr LEFT JOIN templates t ON sr.template_id = t.id WHERE sr.id = ? AND sr.deleted_at IS NULL`, id).
			Scan(&item.ID, &item.TemplateID, &item.TemplateName, &item.FieldsData, &item.ImageURL, &item.IP, &item.CreatedAt)
		if err == sql.ErrNoRows {
			Error(c, http.StatusNotFound, 40003, "资源不存在")
			return
		}
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Success(c, item)
	}
}

func DeleteRecordHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		result, err := db.Exec("UPDATE signing_records SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL", id)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		if rows, _ := result.RowsAffected(); rows == 0 {
			Error(c, http.StatusNotFound, 40003, "资源不存在")
			return
		}
		Success(c, nil)
	}
}
