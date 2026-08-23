package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"inkflow-go/internal/core"
	"inkflow-go/internal/models"
)

type saveFontOutcome struct {
	Font      *models.Font
	Duplicate bool
	Reason    string
}

// saveFont persists a single uploaded font file and returns the outcome.
func saveFont(db *sql.DB, fontsDir string, fh *multipart.FileHeader) saveFontOutcome {
	file, err := fh.Open()
	if err != nil {
		return saveFontOutcome{Reason: "无法读取文件"}
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".ttf" && ext != ".otf" && ext != ".woff" && ext != ".woff2" {
		return saveFontOutcome{Reason: "文件格式不支持"}
	}

	tmpPath := filepath.Join(fontsDir, "_tmp_"+fh.Filename)
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return saveFontOutcome{Reason: "文件写入失败"}
	}
	if _, err := io.Copy(tmp, file); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return saveFontOutcome{Reason: "文件写入失败"}
	}
	tmp.Close()
	defer os.Remove(tmpPath)

	hash, err := core.HashFile(tmpPath)
	if err != nil {
		return saveFontOutcome{Reason: "文件写入失败"}
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM fonts WHERE file_hash = ? AND deleted_at IS NULL", hash).Scan(&count)
	if count > 0 {
		return saveFontOutcome{Duplicate: true, Reason: "字体已存在"}
	}

	displayName, nameErr := core.ReadFontName(tmpPath)
	if nameErr != nil || strings.TrimSpace(displayName) == "" {
		displayName = strings.TrimSuffix(fh.Filename, ext)
	}

	fileName := hash + ext
	destPath := filepath.Join(fontsDir, fileName)
	if err := os.Rename(tmpPath, destPath); err != nil {
		return saveFontOutcome{Reason: "字体文件移动失败"}
	}

	result, err := db.Exec("INSERT INTO fonts (filename, display_name, original_filename, file_hash) VALUES (?, ?, ?, ?)",
		fileName, displayName, fh.Filename, hash)
	if err != nil {
		os.Remove(destPath)
		return saveFontOutcome{Reason: "数据库错误"}
	}
	id, _ := result.LastInsertId()
	return saveFontOutcome{Font: &models.Font{
		ID:               id,
		Filename:         fileName,
		DisplayName:      displayName,
		OriginalFilename: fh.Filename,
		FileHash:         hash,
	}}
}

func ListFontsHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query("SELECT id, filename, COALESCE(display_name,''), COALESCE(original_filename,''), COALESCE(file_hash,''), created_at FROM fonts WHERE deleted_at IS NULL ORDER BY created_at DESC")
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		defer rows.Close()

		items := []models.Font{}
		for rows.Next() {
			var item models.Font
			rows.Scan(&item.ID, &item.Filename, &item.DisplayName, &item.OriginalFilename, &item.FileHash, &item.CreatedAt)
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Success(c, items)
	}
}

func CreateFontHandler(db *sql.DB, fontsDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header, err := c.FormFile("file")
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		out := saveFont(db, fontsDir, header)
		switch {
		case out.Font != nil:
			Success(c, gin.H{"id": out.Font.ID})
		case out.Duplicate:
			Error(c, http.StatusBadRequest, 40001, "字体已存在")
		case out.Reason == "文件格式不支持":
			Error(c, http.StatusBadRequest, 40002, out.Reason)
		default:
			Error(c, http.StatusInternalServerError, 50002, out.Reason)
		}
	}
}

type fontBatchItem struct {
	ID               int64  `json:"id,omitempty"`
	Filename         string `json:"filename,omitempty"`
	DisplayName      string `json:"display_name,omitempty"`
	OriginalFilename string `json:"original_filename"`
	Reason           string `json:"reason,omitempty"`
}

type fontBatchReport struct {
	Uploaded   []fontBatchItem `json:"uploaded"`
	Duplicates []fontBatchItem `json:"duplicates"`
	Failed     []fontBatchItem `json:"failed"`
}

func CreateFontsBatchHandler(db *sql.DB, fontsDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := readFontBatchLimit(db)

		form, err := c.MultipartForm()
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		files := form.File["files"]
		if len(files) == 0 {
			Error(c, http.StatusBadRequest, 40001, "未选择任何文件")
			return
		}
		if limit > 0 && len(files) > limit {
			Error(c, http.StatusBadRequest, 40001, fmt.Sprintf("单次批量导入最多 %d 个文件", limit))
			return
		}

		report := fontBatchReport{Uploaded: []fontBatchItem{}, Duplicates: []fontBatchItem{}, Failed: []fontBatchItem{}}
		for _, fh := range files {
			out := saveFont(db, fontsDir, fh)
			switch {
			case out.Font != nil:
				report.Uploaded = append(report.Uploaded, fontBatchItem{
					ID:          out.Font.ID,
					Filename:    out.Font.Filename,
					DisplayName: out.Font.DisplayName,
				})
			case out.Duplicate:
				report.Duplicates = append(report.Duplicates, fontBatchItem{OriginalFilename: fh.Filename})
			default:
				report.Failed = append(report.Failed, fontBatchItem{OriginalFilename: fh.Filename, Reason: out.Reason})
			}
		}
		Success(c, report)
	}
}

func UpdateFontHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var req struct {
			DisplayName string `json:"display_name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		result, err := db.Exec("UPDATE fonts SET display_name = ? WHERE id = ? AND deleted_at IS NULL", req.DisplayName, id)
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

func DeleteFontHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		result, err := db.Exec("UPDATE fonts SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL", id)
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
