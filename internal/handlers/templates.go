package handlers

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"inkflow-go/internal/core"
	"inkflow-go/internal/models"
)

func ListTemplatesHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		keyword := c.Query("keyword")
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
		offset := (page - 1) * pageSize

		var total int64
		countSQL := "SELECT COUNT(*) FROM templates WHERE deleted_at IS NULL"
		args := []interface{}{}
		if keyword != "" {
			countSQL += " AND name LIKE ?"
			args = append(args, "%"+keyword+"%")
		}
		db.QueryRow(countSQL, args...).Scan(&total)

		dataSQL := "SELECT id, name, created_at FROM templates WHERE deleted_at IS NULL"
		if keyword != "" {
			dataSQL += " AND name LIKE ?"
		}
		dataSQL += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
		queryArgs := make([]interface{}, len(args))
		copy(queryArgs, args)
		queryArgs = append(queryArgs, pageSize, offset)

		rows, err := db.Query(dataSQL, queryArgs...)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		defer rows.Close()

		items := []models.TemplateListItem{}
		for rows.Next() {
			var item models.TemplateListItem
			rows.Scan(&item.ID, &item.Name, &item.CreatedAt)
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Page(c, items, total, page, pageSize)
	}
}

func CreateTemplateHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name       string                     `json:"name"`
			Handwriting *models.HandwritingConfig `json:"handwriting"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		if req.Name == "" {
			req.Name = "未命名模板"
		}
		tx, err := db.Begin()
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		defer tx.Rollback()

		result, err := tx.Exec("INSERT INTO templates (name) VALUES (?)", req.Name)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		id, _ := result.LastInsertId()

		hw := models.DefaultHandwriting()
		if req.Handwriting != nil {
			hw = hw.Merge(*req.Handwriting)
		}
		if _, err := tx.Exec(`INSERT INTO template_handwriting
			(template_id, font_family, paper_enabled, paper_opacity, fiber_count, dot_count,
			 global_tilt, baseline_drift, char_jitter, char_rotation,
			 ink_opacity_min, ink_opacity_max, char_spacing,
			 ink_spots_enabled, ink_spots_chance, ink_spots_max,
			 shadow_blur, checkbox_enabled)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, hw.FontFamily,
			boolToInt(hw.PaperEnabled), hw.PaperOpacity, hw.FiberCount, hw.DotCount,
			hw.GlobalTilt, hw.BaselineDrift, hw.CharJitter, hw.CharRotation,
			hw.InkOpacityMin, hw.InkOpacityMax, hw.CharSpacing,
			boolToInt(hw.InkSpotsEnabled), hw.InkSpotsChance, hw.InkSpotsMax,
			hw.ShadowBlur, boolToInt(hw.CheckboxEnabled)); err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}

		if err := tx.Commit(); err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Success(c, gin.H{"id": id})
	}
}

func GetTemplateHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var t models.Template
		err = db.QueryRow("SELECT id, name, COALESCE(bg_image,''), width, height, created_at FROM templates WHERE id = ? AND deleted_at IS NULL", id).
			Scan(&t.ID, &t.Name, &t.BgImage, &t.Width, &t.Height, &t.CreatedAt)
		if err == sql.ErrNoRows {
			Error(c, http.StatusNotFound, 40003, "资源不存在")
			return
		}
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}

		result := models.TemplateWithHandwriting{Template: t}

		result.Controls = queryControls(db, id)
		if result.Controls == nil {
			result.Controls = []models.Control{}
		}

		result.Rules = queryRules(db, id)

		result.Handwriting = queryHandwriting(db, id)

		Success(c, result)
	}
}

func UpdateTemplateHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var req struct {
			Name        *string                    `json:"name"`
			BgImage     *string                    `json:"bg_image"`
			Width       *int                       `json:"width"`
			Height      *int                       `json:"height"`
			Controls    *[]models.ControlRow       `json:"controls"`
			Rules       *[]models.RuleRow          `json:"rules"`
			Handwriting *models.HandwritingConfig  `json:"handwriting"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}

		tx, err := db.Begin()
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		defer tx.Rollback()

		setClauses := []string{}
		args := []interface{}{}
		if req.Name != nil {
			setClauses = append(setClauses, "name=?")
			args = append(args, *req.Name)
		}
		if req.BgImage != nil {
			setClauses = append(setClauses, "bg_image=?")
			args = append(args, *req.BgImage)
		}
		if req.Width != nil {
			setClauses = append(setClauses, "width=?")
			args = append(args, *req.Width)
		}
		if req.Height != nil {
			setClauses = append(setClauses, "height=?")
			args = append(args, *req.Height)
		}
		if len(setClauses) > 0 {
			sqlStr := "UPDATE templates SET " + strings.Join(setClauses, ", ") + " WHERE id=? AND deleted_at IS NULL"
			args = append(args, id)
			if _, err := tx.Exec(sqlStr, args...); err != nil {
				Error(c, http.StatusInternalServerError, 50001, "数据库错误")
				return
			}
		}

		if req.Controls != nil {
			if _, err := tx.Exec("DELETE FROM template_controls WHERE template_id=?", id); err != nil {
				Error(c, http.StatusInternalServerError, 50001, "数据库错误")
				return
			}
			for i, ctl := range *req.Controls {
				ctl.TemplateID = id
				ctl.SortOrder = i
				if _, err := tx.Exec(`INSERT INTO template_controls
					(id, template_id, label, type, x, y, width, height, font_size, font_family, required, preview_text, check_size, sort_order)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					ctl.ID, ctl.TemplateID, ctl.Label, ctl.Type, ctl.X, ctl.Y, ctl.Width, ctl.Height,
					ctl.FontSize, ctl.FontFamily, boolToInt(&ctl.Required), ctl.PreviewText, ctl.CheckSize, ctl.SortOrder); err != nil {
					Error(c, http.StatusInternalServerError, 50001, "数据库错误")
					return
				}
			}
		}

		if req.Rules != nil {
			if _, err := tx.Exec("DELETE FROM template_rules WHERE template_id=?", id); err != nil {
				Error(c, http.StatusInternalServerError, 50001, "数据库错误")
				return
			}
			for i, rule := range *req.Rules {
				rule.TemplateID = id
				rule.SortOrder = i
				configJSON := "{}"
				if rule.Config != nil {
					b, _ := json.Marshal(rule.Config)
					configJSON = string(b)
				}
				if _, err := tx.Exec(`INSERT INTO template_rules (id, template_id, type, name, target, config_json, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					rule.ID, rule.TemplateID, rule.Type, rule.Name, rule.Target, configJSON, rule.SortOrder); err != nil {
					Error(c, http.StatusInternalServerError, 50001, "数据库错误")
					return
				}
			}
		}

		if req.Handwriting != nil {
			// Read existing handwriting and merge with incoming values
			existing := models.DefaultHandwriting()
			var ef string
			var ep float64; var epo float64; var efc, edc float64
			var egt, ebd, ecj, ecr, eimn, eimx, ecs, eisc, esc, esm, esb, ecb float64
			err := tx.QueryRow(`SELECT COALESCE(font_family,'sans-serif'), paper_enabled, paper_opacity, fiber_count, dot_count,
				global_tilt, baseline_drift, char_jitter, char_rotation,
				ink_opacity_min, ink_opacity_max, char_spacing,
				ink_spots_enabled, ink_spots_chance, ink_spots_max,
				shadow_blur, checkbox_enabled
				FROM template_handwriting WHERE template_id=?`, id).Scan(
				&ef, &ep, &epo, &efc, &edc,
				&egt, &ebd, &ecj, &ecr, &eimn, &eimx, &ecs, &eisc, &esc, &esm, &esb, &ecb)
			if err == nil {
				existing.FontFamily = models.SP(ef)
				bp := ep != 0; existing.PaperEnabled = &bp
				existing.PaperOpacity = models.FP(epo)
				existing.FiberCount = models.IP(int(efc))
				existing.DotCount = models.IP(int(edc))
				existing.GlobalTilt = models.FP(egt)
				existing.BaselineDrift = models.FP(ebd)
				existing.CharJitter = models.FP(ecj)
				existing.CharRotation = models.FP(ecr)
				existing.InkOpacityMin = models.FP(eimn)
				existing.InkOpacityMax = models.FP(eimx)
				existing.CharSpacing = models.FP(ecs)
				existing.InkSpotsEnabled = boolPtr(int(eisc))
				existing.InkSpotsChance = models.FP(esc)
				existing.InkSpotsMax = models.IP(int(esm))
				existing.ShadowBlur = models.FP(esb)
				existing.CheckboxEnabled = boolPtr(int(ecb))
			}
			hw := existing.Merge(*req.Handwriting)
			// Try UPDATE first; if no row exists, INSERT
			res, updErr := tx.Exec(`UPDATE template_handwriting SET
				font_family=?, paper_enabled=?, paper_opacity=?, fiber_count=?, dot_count=?,
				global_tilt=?, baseline_drift=?, char_jitter=?, char_rotation=?,
				ink_opacity_min=?, ink_opacity_max=?, char_spacing=?,
				ink_spots_enabled=?, ink_spots_chance=?, ink_spots_max=?,
				shadow_blur=?, checkbox_enabled=?, updated_at=CURRENT_TIMESTAMP
				WHERE template_id=?`,
				hw.FontFamily,
				boolToInt(hw.PaperEnabled), hw.PaperOpacity, hw.FiberCount, hw.DotCount,
				hw.GlobalTilt, hw.BaselineDrift, hw.CharJitter, hw.CharRotation,
				hw.InkOpacityMin, hw.InkOpacityMax, hw.CharSpacing,
				boolToInt(hw.InkSpotsEnabled), hw.InkSpotsChance, hw.InkSpotsMax,
				hw.ShadowBlur, boolToInt(hw.CheckboxEnabled), id)
			if updErr != nil {
				Error(c, http.StatusInternalServerError, 50001, "数据库错误")
				return
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				if _, err := tx.Exec(`INSERT INTO template_handwriting
					(template_id, font_family, paper_enabled, paper_opacity, fiber_count, dot_count,
					 global_tilt, baseline_drift, char_jitter, char_rotation,
					 ink_opacity_min, ink_opacity_max, char_spacing,
					 ink_spots_enabled, ink_spots_chance, ink_spots_max,
					 shadow_blur, checkbox_enabled)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					id, hw.FontFamily,
					boolToInt(hw.PaperEnabled), hw.PaperOpacity, hw.FiberCount, hw.DotCount,
					hw.GlobalTilt, hw.BaselineDrift, hw.CharJitter, hw.CharRotation,
					hw.InkOpacityMin, hw.InkOpacityMax, hw.CharSpacing,
					boolToInt(hw.InkSpotsEnabled), hw.InkSpotsChance, hw.InkSpotsMax,
					hw.ShadowBlur, boolToInt(hw.CheckboxEnabled)); err != nil {
					Error(c, http.StatusInternalServerError, 50001, "数据库错误")
					return
				}
			}
		}

		if err := tx.Commit(); err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Success(c, nil)
	}
}

func DeleteTemplateHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		result, err := db.Exec("UPDATE templates SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL", id)
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

func UploadTemplateFileHandler(db *sql.DB, bgImageDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var exists int
		db.QueryRow("SELECT COUNT(*) FROM templates WHERE id = ? AND deleted_at IS NULL", id).Scan(&exists)
		if exists == 0 {
			Error(c, http.StatusNotFound, 40003, "资源不存在")
			return
		}
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".pdf" {
			Error(c, http.StatusBadRequest, 40002, "文件格式不支持")
			return
		}

		dir := filepath.Join(bgImageDir, fmt.Sprintf("%d", id))
		os.MkdirAll(dir, 0755)

		if ext == ".pdf" {
			tmpPath := filepath.Join(dir, "_tmp_upload.pdf")
			tmp, err := os.Create(tmpPath)
			if err != nil {
				Error(c, http.StatusInternalServerError, 50002, "文件写入失败")
				return
			}
			if _, err := io.Copy(tmp, file); err != nil {
				tmp.Close()
				os.Remove(tmpPath)
				Error(c, http.StatusInternalServerError, 50002, "文件写入失败")
				return
			}
			tmp.Close()

			outDir := filepath.Join(dir, "pages")
			os.MkdirAll(outDir, 0755)
			images, err := core.PDFToImages(tmpPath, outDir)
			os.Remove(tmpPath)
			if err != nil {
				Error(c, http.StatusInternalServerError, 50003, "PDF 转换失败")
				return
			}
			if len(images) == 0 {
				Error(c, http.StatusInternalServerError, 50003, "PDF 转换失败: 无页面")
				return
			}

			bgRelPath := fmt.Sprintf("bg_images/%d/pages/page_1.png", id)
			if _, err := db.Exec("UPDATE templates SET bg_image = ? WHERE id = ?", bgRelPath, id); err != nil {
				os.RemoveAll(outDir)
				Error(c, http.StatusInternalServerError, 50001, "数据库错误")
				return
			}
			Success(c, gin.H{"bg_image": bgRelPath, "all_pages": images})
		} else {
			outPath := filepath.Join(dir, fmt.Sprintf("bg%s", ext))
			out, err := os.Create(outPath)
			if err != nil {
				Error(c, http.StatusInternalServerError, 50002, "文件写入失败")
				return
			}
			if _, err := io.Copy(out, file); err != nil {
				out.Close()
				os.Remove(outPath)
				Error(c, http.StatusInternalServerError, 50002, "文件写入失败")
				return
			}
			out.Close()

			bgRelPath := fmt.Sprintf("bg_images/%d/bg%s", id, ext)
			if _, err := db.Exec("UPDATE templates SET bg_image = ? WHERE id = ?", bgRelPath, id); err != nil {
				os.Remove(outPath)
				Error(c, http.StatusInternalServerError, 50001, "数据库错误")
				return
			}
			Success(c, gin.H{"bg_image": bgRelPath})
		}
	}
}

func ExportTemplateHandler(db *sql.DB, bgImageDir string, fontsDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var t models.Template
		err = db.QueryRow("SELECT id, name, COALESCE(bg_image,''), width, height, created_at FROM templates WHERE id = ? AND deleted_at IS NULL", id).
			Scan(&t.ID, &t.Name, &t.BgImage, &t.Width, &t.Height, &t.CreatedAt)
		if err == sql.ErrNoRows {
			Error(c, http.StatusNotFound, 40003, "资源不存在")
			return
		}
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}

		controls := queryControls(db, id)
		rules := queryRules(db, id)
		hw := queryHandwriting(db, id)

		buf := new(bytes.Buffer)
		w := zip.NewWriter(buf)

		exportData := map[string]interface{}{
			"export_version": "2.0",
			"exported_at":    time.Now().UTC().Format(time.RFC3339),
			"template": map[string]interface{}{
				"name":      t.Name,
				"bg_image":  t.BgImage,
				"width":     t.Width,
				"height":    t.Height,
				"controls":  controls,
				"rules":     rules,
			},
		}
		if hw != nil {
			exportData["handwriting"] = hw
		}
		exportJSON, _ := json.MarshalIndent(exportData, "", "  ")
		f, _ := w.Create("template.json")
		f.Write(exportJSON)

		if t.BgImage != "" {
			bgPath := resolveBgPath(bgImageDir, t.BgImage)
			if bgPath != "" {
				if bgData, err := os.ReadFile(bgPath); err == nil {
					bgEntry, _ := w.Create("bg_image/" + filepath.Base(t.BgImage))
					bgEntry.Write(bgData)
				}
			}
		}

		fontFamilies := make(map[string]bool)
		for _, ctl := range controls {
			if ctl.FontFamily != "" {
				fontFamilies[ctl.FontFamily] = true
			}
		}

		if len(fontFamilies) > 0 {
			var fontsMeta []map[string]interface{}
			rows, err := db.Query("SELECT id, filename, display_name, original_filename FROM fonts WHERE deleted_at IS NULL")
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var f models.Font
					if err := rows.Scan(&f.ID, &f.Filename, &f.DisplayName, &f.OriginalFilename); err != nil {
						continue
					}
					if !fontFamilies[f.DisplayName] {
						continue
					}
					fontPath := filepath.Join(fontsDir, f.Filename)
					if fontData, err := os.ReadFile(fontPath); err == nil {
						fontEntry, _ := w.Create("fonts/" + f.Filename)
						fontEntry.Write(fontData)
					}
					fontsMeta = append(fontsMeta, map[string]interface{}{
						"id":                f.ID,
						"filename":          f.Filename,
						"display_name":      f.DisplayName,
						"original_filename": f.OriginalFilename,
					})
				}
			}
			if len(fontsMeta) > 0 {
				fontsJSON, _ := json.MarshalIndent(fontsMeta, "", "  ")
				ff, _ := w.Create("fonts.json")
				ff.Write(fontsJSON)
			}
		}

		w.Close()

	asciiName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, t.Name)
	if asciiName == "" {
		asciiName = "template"
	}
	asciiFilename := fmt.Sprintf("%s_%d.zip", asciiName, id)

	utf8Filename := fmt.Sprintf("%s_%d.zip", t.Name, id)
	encodedName := (&url.URL{Path: utf8Filename}).RequestURI()

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, asciiFilename, encodedName))
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
	}
}

func ImportTemplateHandler(db *sql.DB, bgImageDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		defer file.Close()

		if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
			Error(c, http.StatusBadRequest, 40002, "仅支持 ZIP 文件导入")
			return
		}

		zipReader, err := zip.NewReader(file, header.Size)
		if err != nil {
			Error(c, http.StatusBadRequest, 40002, "无效的 ZIP 文件")
			return
		}

		var templateJSON []byte
		var bgImageData []byte
		var bgImageName string
		var fontsData map[string][]byte
		var fontsMetaJSON []byte

		for _, f := range zipReader.File {
			if f.Name == "template.json" {
				rc, err := f.Open()
				if err != nil {
					Error(c, http.StatusInternalServerError, 50002, "ZIP 读取失败")
					return
				}
				templateJSON, _ = io.ReadAll(rc)
				rc.Close()
			} else if strings.HasPrefix(f.Name, "bg_image/") && !strings.HasSuffix(f.Name, "/") {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				bgImageData, _ = io.ReadAll(rc)
				rc.Close()
				bgImageName = filepath.Base(f.Name)
			} else if strings.HasPrefix(f.Name, "fonts/") && !strings.HasSuffix(f.Name, "/") {
				if fontsData == nil {
					fontsData = make(map[string][]byte)
				}
				rc, err := f.Open()
				if err != nil {
					continue
				}
				data, _ := io.ReadAll(rc)
				rc.Close()
				fontsData[filepath.Base(f.Name)] = data
			} else if f.Name == "fonts.json" {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				fontsMetaJSON, _ = io.ReadAll(rc)
				rc.Close()
			}
		}

		if templateJSON == nil {
			Error(c, http.StatusBadRequest, 40002, "ZIP 中未找到 template.json")
			return
		}

		var exportData map[string]interface{}
		if err := json.Unmarshal(templateJSON, &exportData); err != nil {
			Error(c, http.StatusBadRequest, 40002, "template.json 格式无效")
			return
		}

		tmplData, ok := exportData["template"].(map[string]interface{})
		if !ok {
			tmplData = exportData
		}

		name, _ := tmplData["name"].(string)
		if name == "" {
			name = "导入的模板"
		}
		width := 800
		if v, ok := tmplData["width"].(float64); ok {
			width = int(v)
		}
		height := 1000
		if v, ok := tmplData["height"].(float64); ok {
			height = int(v)
		}

		var controls []models.Control
		if controlsRaw, ok := tmplData["controls"].([]interface{}); ok {
			for _, c := range controlsRaw {
				if cm, ok := c.(map[string]interface{}); ok {
					ctl := models.Control{}
					if v, ok := cm["id"].(string); ok {
						ctl.ID = v
					}
					if v, ok := cm["label"].(string); ok {
						ctl.Label = v
					}
					if v, ok := cm["type"].(string); ok {
						ctl.Type = v
					}
					if v, ok := cm["x"].(float64); ok {
						ctl.X = v
					}
					if v, ok := cm["y"].(float64); ok {
						ctl.Y = v
					}
					if v, ok := cm["width"].(float64); ok {
						ctl.Width = v
					}
					if v, ok := cm["height"].(float64); ok {
						ctl.Height = v
					}
					if v, ok := cm["font_size"].(float64); ok {
						ctl.FontSize = int(v)
					}
					if v, ok := cm["font_family"].(string); ok {
						ctl.FontFamily = v
					}
					if v, ok := cm["required"].(bool); ok {
						ctl.Required = v
					}
					if v, ok := cm["preview_text"].(string); ok {
						ctl.PreviewText = v
					}
					if v, ok := cm["check_size"].(float64); ok {
						ctl.CheckSize = int(v)
					}
					controls = append(controls, ctl)
				}
			}
		}

		type ImportedRule struct {
			ID         string      `json:"id"`
			Type       string      `json:"type"`
			Name       string      `json:"name"`
			Target     string      `json:"target"`
			ConfigJSON string      `json:"-"`
			Config     interface{} `json:"config"`
		}
		var rules []ImportedRule
		if rulesRaw, ok := tmplData["rules"].([]interface{}); ok {
			for _, r := range rulesRaw {
				if rm, ok := r.(map[string]interface{}); ok {
					ir := ImportedRule{}
					if v, ok := rm["id"].(string); ok {
						ir.ID = v
					}
					if v, ok := rm["type"].(string); ok {
						ir.Type = v
					}
					if v, ok := rm["name"].(string); ok {
						ir.Name = v
					}
					if v, ok := rm["target"].(string); ok {
						ir.Target = v
					}
					if cfg, ok := rm["config"]; ok {
						cfgJSON, _ := json.Marshal(cfg)
						ir.ConfigJSON = string(cfgJSON)
						ir.Config = cfg
					}
					rules = append(rules, ir)
				}
			}
		}

		tx, err := db.Begin()
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		defer tx.Rollback()

		var bgImageURL string
		if len(bgImageData) > 0 && bgImageName != "" {
			bgDir := filepath.Join(bgImageDir, "imported")
			os.MkdirAll(bgDir, 0755)
			bgPath := filepath.Join(bgDir, bgImageName)
			if err := os.WriteFile(bgPath, bgImageData, 0644); err == nil {
				bgImageURL = "bg_images/imported/" + bgImageName
			}
		}

		var tmplID int64
		// Check if template with same name already exists
		var existingID int64
		err = tx.QueryRow("SELECT id FROM templates WHERE name = ? AND deleted_at IS NULL", name).Scan(&existingID)
		if err == sql.ErrNoRows {
			// Template name doesn't exist, create new
			result, err := tx.Exec("INSERT INTO templates (name, width, height, bg_image) VALUES (?, ?, ?, ?)",
				name, width, height, bgImageURL)
			if err != nil {
				Error(c, http.StatusInternalServerError, 50001, "数据库错误")
				return
			}
			tmplID, _ = result.LastInsertId()
		} else if err == nil {
			// Template name exists, update existing template
			tmplID = existingID
			if _, err := tx.Exec("UPDATE templates SET width=?, height=?, bg_image=? WHERE id=?",
				width, height, bgImageURL, tmplID); err != nil {
				Error(c, http.StatusInternalServerError, 50001, "数据库错误")
				return
			}
			// Clear existing controls, rules, and handwriting for this template
			tx.Exec("DELETE FROM template_controls WHERE template_id=?", tmplID)
			tx.Exec("DELETE FROM template_rules WHERE template_id=?", tmplID)
			tx.Exec("DELETE FROM template_handwriting WHERE template_id=?", tmplID)
		} else {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}

		hw := models.DefaultHandwriting()
		if hwRaw, ok := exportData["handwriting"].(map[string]interface{}); ok {
			if v, ok := hwRaw["font_family"].(string); ok {
				hw.FontFamily = &v
			}
			if v, ok := hwRaw["paper_enabled"].(bool); ok {
				hw.PaperEnabled = &v
			}
			if v, ok := hwRaw["paper_opacity"].(float64); ok {
				hw.PaperOpacity = &v
			}
			if v, ok := hwRaw["fiber_count"].(float64); ok {
				n := int(v)
				hw.FiberCount = &n
			}
			if v, ok := hwRaw["dot_count"].(float64); ok {
				n := int(v)
				hw.DotCount = &n
			}
			if v, ok := hwRaw["global_tilt"].(float64); ok {
				hw.GlobalTilt = &v
			}
			if v, ok := hwRaw["baseline_drift"].(float64); ok {
				hw.BaselineDrift = &v
			}
			if v, ok := hwRaw["char_jitter"].(float64); ok {
				hw.CharJitter = &v
			}
			if v, ok := hwRaw["char_rotation"].(float64); ok {
				hw.CharRotation = &v
			}
			if v, ok := hwRaw["ink_opacity_min"].(float64); ok {
				hw.InkOpacityMin = &v
			}
			if v, ok := hwRaw["ink_opacity_max"].(float64); ok {
				hw.InkOpacityMax = &v
			}
			if v, ok := hwRaw["char_spacing"].(float64); ok {
				hw.CharSpacing = &v
			}
			if v, ok := hwRaw["ink_spots_enabled"].(bool); ok {
				hw.InkSpotsEnabled = &v
			}
			if v, ok := hwRaw["ink_spots_chance"].(float64); ok {
				hw.InkSpotsChance = &v
			}
			if v, ok := hwRaw["ink_spots_max"].(float64); ok {
				n := int(v)
				hw.InkSpotsMax = &n
			}
			if v, ok := hwRaw["shadow_blur"].(float64); ok {
				hw.ShadowBlur = &v
			}
			if v, ok := hwRaw["checkbox_enabled"].(bool); ok {
				hw.CheckboxEnabled = &v
			}
		}
		if _, err := tx.Exec(`INSERT INTO template_handwriting
			(template_id, font_family, paper_enabled, paper_opacity, fiber_count, dot_count,
			 global_tilt, baseline_drift, char_jitter, char_rotation,
			 ink_opacity_min, ink_opacity_max, char_spacing,
			 ink_spots_enabled, ink_spots_chance, ink_spots_max,
			 shadow_blur, checkbox_enabled)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			tmplID, hw.FontFamily,
			boolToInt(hw.PaperEnabled), hw.PaperOpacity, hw.FiberCount, hw.DotCount,
			hw.GlobalTilt, hw.BaselineDrift, hw.CharJitter, hw.CharRotation,
			hw.InkOpacityMin, hw.InkOpacityMax, hw.CharSpacing,
			boolToInt(hw.InkSpotsEnabled), hw.InkSpotsChance, hw.InkSpotsMax,
			hw.ShadowBlur, boolToInt(hw.CheckboxEnabled)); err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}

		for i, ctl := range controls {
			// Use UPSERT: check if control with same ID exists for this template
			var ctlExists int
			err := tx.QueryRow("SELECT COUNT(*) FROM template_controls WHERE id=? AND template_id=?", ctl.ID, tmplID).Scan(&ctlExists)
			if err != nil {
				Error(c, http.StatusInternalServerError, 50001, "数据库错误")
				return
			}
			if ctlExists > 0 {
				// Update existing control
				if _, err := tx.Exec(`UPDATE template_controls
					SET label=?, type=?, x=?, y=?, width=?, height=?, font_size=?, font_family=?, required=?, preview_text=?, check_size=?, sort_order=?
					WHERE id=? AND template_id=?`,
					ctl.Label, ctl.Type, ctl.X, ctl.Y, ctl.Width, ctl.Height,
					ctl.FontSize, ctl.FontFamily, boolToInt(&ctl.Required), ctl.PreviewText, ctl.CheckSize, i,
					ctl.ID, tmplID); err != nil {
					Error(c, http.StatusInternalServerError, 50001, "数据库错误")
					return
				}
			} else {
				// Insert new control
				if _, err := tx.Exec(`INSERT INTO template_controls
					(id, template_id, label, type, x, y, width, height, font_size, font_family, required, preview_text, check_size, sort_order)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					ctl.ID, tmplID, ctl.Label, ctl.Type, ctl.X, ctl.Y, ctl.Width, ctl.Height,
					ctl.FontSize, ctl.FontFamily, boolToInt(&ctl.Required), ctl.PreviewText, ctl.CheckSize, i); err != nil {
					Error(c, http.StatusInternalServerError, 50001, "数据库错误")
					return
				}
			}
		}

		for i, rule := range rules {
			// Use UPSERT: check if rule with same ID exists for this template
			var ruleExists int
			err := tx.QueryRow("SELECT COUNT(*) FROM template_rules WHERE id=? AND template_id=?", rule.ID, tmplID).Scan(&ruleExists)
			if err != nil {
				Error(c, http.StatusInternalServerError, 50001, "数据库错误")
				return
			}
			if ruleExists > 0 {
				// Update existing rule
				if _, err := tx.Exec(`UPDATE template_rules
					SET type=?, name=?, target=?, config_json=?, sort_order=?
					WHERE id=? AND template_id=?`,
					rule.Type, rule.Name, rule.Target, rule.ConfigJSON, i,
					rule.ID, tmplID); err != nil {
					Error(c, http.StatusInternalServerError, 50001, "数据库错误")
					return
				}
			} else {
				// Insert new rule
				if _, err := tx.Exec(`INSERT INTO template_rules
					(id, template_id, type, name, target, config_json, sort_order)
					VALUES (?, ?, ?, ?, ?, ?, ?)`,
					rule.ID, tmplID, rule.Type, rule.Name, rule.Target, rule.ConfigJSON, i); err != nil {
					Error(c, http.StatusInternalServerError, 50001, "数据库错误")
					return
				}
			}
		}

		fontsDir := filepath.Dir(bgImageDir)
		fontsDir = filepath.Join(fontsDir, "fonts")
		if len(fontsData) > 0 {
			os.MkdirAll(fontsDir, 0755)
		}
		var importedFonts []map[string]interface{}
		if len(fontsMetaJSON) > 0 {
			var fontsMeta []map[string]interface{}
			if json.Unmarshal(fontsMetaJSON, &fontsMeta) == nil {
				for _, fm := range fontsMeta {
					filename, _ := fm["filename"].(string)
					displayName, _ := fm["display_name"].(string)
					origName, _ := fm["original_filename"].(string)
					if filename == "" || displayName == "" {
						continue
					}
					if fontData, ok := fontsData[filename]; ok {
						fontPath := filepath.Join(fontsDir, filename)
						os.WriteFile(fontPath, fontData, 0644)
					}
					var existingID int64
					err := tx.QueryRow("SELECT id FROM fonts WHERE display_name=? AND deleted_at IS NULL", displayName).Scan(&existingID)
					if err == sql.ErrNoRows {
						res, err := tx.Exec("INSERT INTO fonts (filename, display_name, original_filename) VALUES (?, ?, ?)",
							filename, displayName, origName)
						if err == nil {
							newID, _ := res.LastInsertId()
							importedFonts = append(importedFonts, map[string]interface{}{
								"id": newID, "display_name": displayName,
							})
						}
					} else if err == nil {
						importedFonts = append(importedFonts, map[string]interface{}{
							"id": existingID, "display_name": displayName,
						})
					}
				}
			}
		}

		if err := tx.Commit(); err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Success(c, gin.H{"id": tmplID, "name": name, "imported_fonts": len(importedFonts)})
	}
}

// resolveBgPath resolves the background image path from DB value and bgImageDir.
// DB stores paths like "bg_images/1/pages/page_1.png", bgImageDir is "./data/bg_images".
func resolveBgPath(bgImageDir, dbPath string) string {
	// Try direct join first (works if dbPath is relative like "1/pages/page_1.png")
	direct := filepath.Join(bgImageDir, dbPath)
	if _, err := os.Stat(direct); err == nil {
		return direct
	}
	// Strip "bg_images/" prefix and join with bgImageDir
	stripped := strings.TrimPrefix(dbPath, "bg_images/")
	if stripped != dbPath {
		alt := filepath.Join(bgImageDir, stripped)
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	}
	// Try absolute path (dbPath might be stored as absolute)
	if _, err := os.Stat(dbPath); err == nil {
		return dbPath
	}
	return ""
}

func queryControls(db *sql.DB, templateID int64) []models.Control {
	rows, err := db.Query(`SELECT id, label, type, x, y, width, height, font_size, font_family, required, preview_text, check_size
		FROM template_controls WHERE template_id=? ORDER BY sort_order`, templateID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var controls []models.Control
	for rows.Next() {
		var c models.Control
		var required int
		if err := rows.Scan(&c.ID, &c.Label, &c.Type, &c.X, &c.Y, &c.Width, &c.Height, &c.FontSize, &c.FontFamily, &required, &c.PreviewText, &c.CheckSize); err != nil {
			continue
		}
		c.Required = required == 1
		controls = append(controls, c)
	}
	return controls
}

func queryRules(db *sql.DB, templateID int64) []models.RuleRow {
	rows, err := db.Query(`SELECT id, type, name, target, config_json
		FROM template_rules WHERE template_id=? ORDER BY sort_order`, templateID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var rules []models.RuleRow
	for rows.Next() {
		var r models.RuleRow
		if err := rows.Scan(&r.ID, &r.Type, &r.Name, &r.Target, &r.ConfigJSON); err != nil {
			continue
		}
		var config interface{}
		if r.ConfigJSON != "" && r.ConfigJSON != "{}" {
			json.Unmarshal([]byte(r.ConfigJSON), &config)
		}
		r.Config = config
		rules = append(rules, r)
	}
	return rules
}

func queryHandwriting(db *sql.DB, templateID int64) *models.HandwritingConfig {
	var hw models.HandwritingConfig
	var paperEnabled, inkSpotsEnabled, checkboxEnabled int
	err := db.QueryRow(`SELECT font_family, paper_enabled, paper_opacity, fiber_count, dot_count,
		global_tilt, baseline_drift, char_jitter, char_rotation,
		ink_opacity_min, ink_opacity_max, char_spacing,
		ink_spots_enabled, ink_spots_chance, ink_spots_max,
		shadow_blur, checkbox_enabled
		FROM template_handwriting WHERE template_id=?`, templateID).
		Scan(&hw.FontFamily, &paperEnabled, &hw.PaperOpacity, &hw.FiberCount, &hw.DotCount,
			&hw.GlobalTilt, &hw.BaselineDrift, &hw.CharJitter, &hw.CharRotation,
			&hw.InkOpacityMin, &hw.InkOpacityMax, &hw.CharSpacing,
			&inkSpotsEnabled, &hw.InkSpotsChance, &hw.InkSpotsMax,
			&hw.ShadowBlur, &checkboxEnabled)
	if err != nil {
		return nil
	}
	t1 := paperEnabled == 1
	t2 := inkSpotsEnabled == 1
	t3 := checkboxEnabled == 1
	hw.PaperEnabled = &t1
	hw.InkSpotsEnabled = &t2
	hw.CheckboxEnabled = &t3
	return &hw
}

func boolToInt(b *bool) int {
	if b != nil && *b {
		return 1
	}
	return 0
}

func boolPtr(v int) *bool {
	b := v != 0
	return &b
}
