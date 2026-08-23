package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"inkflow-go/internal/core"
	"inkflow-go/internal/models"
)

type SignRequest struct {
	FieldsData    map[string]interface{}   `json:"fields_data"`
	TextLayerData string                   `json:"text_layer_data"`
	Handwriting   *models.HandwritingConfig `json:"handwriting,omitempty"`
}

func mergeHandwriting(db *sql.DB, templateID int64, signOverride *models.HandwritingConfig) models.HandwritingConfig {
	merged := models.DefaultHandwriting()

	var globalValue string
	err := db.QueryRow("SELECT value FROM settings WHERE key = 'handwriting'").Scan(&globalValue)
	if err == nil {
		var globalHW models.HandwritingConfig
		if json.Unmarshal([]byte(globalValue), &globalHW) == nil {
			merged = merged.Merge(globalHW)
		}
	}

	var hw models.HandwritingConfig
	var paperEnabled, inkSpotsEnabled, checkboxEnabled int
	err = db.QueryRow(`SELECT font_family, paper_enabled, paper_opacity, fiber_count, dot_count,
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
	if err == nil {
		t1 := paperEnabled == 1
		t2 := inkSpotsEnabled == 1
		t3 := checkboxEnabled == 1
		hw.PaperEnabled = &t1
		hw.InkSpotsEnabled = &t2
		hw.CheckboxEnabled = &t3
		merged = merged.Merge(hw)
	}

	if signOverride != nil {
		merged = merged.Merge(*signOverride)
	}

	return merged
}

func SignHandler(db *sql.DB, bgImageDir string, outputDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var req SignRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var name string
		var bgImage string
		err = db.QueryRow("SELECT name, COALESCE(bg_image,'') FROM templates WHERE id = ? AND deleted_at IS NULL", id).Scan(&name, &bgImage)
		if err == sql.ErrNoRows {
			Error(c, http.StatusNotFound, 40003, "资源不存在")
			return
		}
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}

		if bgImage == "" {
			Error(c, http.StatusBadRequest, 40002, "模板无背景图，请先上传")
			return
		}

		fieldsJSON, _ := json.Marshal(req.FieldsData)
		ip := c.ClientIP()

		bgPath := filepath.Join(bgImageDir, bgImage)
		if _, err := os.Stat(bgPath); os.IsNotExist(err) {
			altPath := filepath.Join(bgImageDir, strings.TrimPrefix(bgImage, "bg_images/"))
			if _, err2 := os.Stat(altPath); os.IsNotExist(err2) {
				Error(c, http.StatusInternalServerError, 50003, fmt.Sprintf("背景图文件不存在: %s", bgImage))
				return
			}
			bgPath = altPath
		}

		bgFile, err := os.Open(bgPath)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50003, "背景图读取失败")
			return
		}
		defer bgFile.Close()

		bgImg, err := png.Decode(bgFile)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50003, "背景图解码失败")
			return
		}

		var textLayer image.Image
		if req.TextLayerData != "" && strings.HasPrefix(req.TextLayerData, "data:image/png;base64,") {
			b64Data := strings.TrimPrefix(req.TextLayerData, "data:image/png;base64,")
			imgData, err := base64.StdEncoding.DecodeString(b64Data)
			if err != nil {
				Error(c, http.StatusBadRequest, 40001, "文本图层数据无效")
				return
			}
			textLayer, err = png.Decode(strings.NewReader(string(imgData)))
			if err != nil {
				Error(c, http.StatusBadRequest, 40001, "文本图层解码失败")
				return
			}
		} else {
			textLayer = image.NewRGBA(bgImg.Bounds())
		}

		bgRGBA := image.NewRGBA(bgImg.Bounds())
		for y := bgImg.Bounds().Min.Y; y < bgImg.Bounds().Max.Y; y++ {
			for x := bgImg.Bounds().Min.X; x < bgImg.Bounds().Max.X; x++ {
				bgRGBA.Set(x, y, bgImg.At(x, y))
			}
		}

		textRGBA := image.NewRGBA(bgImg.Bounds())
		for y := textLayer.Bounds().Min.Y; y < textLayer.Bounds().Max.Y; y++ {
			for x := textLayer.Bounds().Min.X; x < textLayer.Bounds().Max.X; x++ {
				textRGBA.Set(x, y, textLayer.At(x, y))
			}
		}

		finalImg := core.Compose(bgRGBA, textRGBA, int64(id))

		os.MkdirAll(outputDir, 0755)
		outFileName := fmt.Sprintf("signed_%d_%s.png", id, strconv.FormatInt(int64(os.Getpid()), 36))
		outPath := filepath.Join(outputDir, outFileName)
		outFile, err := os.Create(outPath)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50002, "输出文件创建失败")
			return
		}
		defer outFile.Close()

		if err := png.Encode(outFile, finalImg); err != nil {
			Error(c, http.StatusInternalServerError, 50002, "图片编码失败")
			return
		}
		outFile.Close()

		imageURL := "/static/output/" + outFileName

		result, err := db.Exec("INSERT INTO signing_records (template_id, fields_data, image_url, ip_address) VALUES (?, ?, ?, ?)",
			id, string(fieldsJSON), imageURL, ip)
		if err != nil {
			os.Remove(outPath)
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		recordID, _ := result.LastInsertId()

		Success(c, gin.H{
			"record_id": recordID,
			"image_url": imageURL,
		})
	}
}
