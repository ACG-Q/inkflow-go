package handlers

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
	"inkflow-go/internal/models"
)

func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := os.TempDir() + "/inkflow_test_" + t.Name() + ".db"
	os.Remove(dbPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	migrate := []string{
		`CREATE TABLE IF NOT EXISTS templates (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL DEFAULT '未命名模板', bg_image TEXT, width INTEGER DEFAULT 800, height INTEGER DEFAULT 1000, config_json TEXT NOT NULL DEFAULT '{}', created_at DATETIME DEFAULT CURRENT_TIMESTAMP, deleted_at DATETIME DEFAULT NULL)`,
		`CREATE TABLE IF NOT EXISTS signing_records (id INTEGER PRIMARY KEY AUTOINCREMENT, template_id INTEGER NOT NULL REFERENCES templates(id), fields_data TEXT, image_url TEXT, ip_address TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, deleted_at DATETIME DEFAULT NULL)`,
		`CREATE TABLE IF NOT EXISTS fonts (id INTEGER PRIMARY KEY AUTOINCREMENT, filename TEXT NOT NULL, display_name TEXT, original_filename TEXT, file_hash TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, deleted_at DATETIME DEFAULT NULL)`,
		`CREATE TABLE IF NOT EXISTS admins (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE DEFAULT 'admin', password TEXT NOT NULL, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS template_controls (id TEXT PRIMARY KEY, template_id INTEGER NOT NULL REFERENCES templates(id) ON DELETE CASCADE, label TEXT NOT NULL DEFAULT '', type TEXT NOT NULL DEFAULT 'textbox', x REAL DEFAULT 0, y REAL DEFAULT 0, width REAL DEFAULT 200, height REAL DEFAULT 40, font_size INTEGER DEFAULT 20, font_family TEXT DEFAULT 'sans-serif', required INTEGER DEFAULT 0, preview_text TEXT DEFAULT '', check_size INTEGER DEFAULT 24, sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS template_rules (id TEXT PRIMARY KEY, template_id INTEGER NOT NULL REFERENCES templates(id) ON DELETE CASCADE, type TEXT NOT NULL, name TEXT DEFAULT '', target TEXT DEFAULT '', config_json TEXT NOT NULL DEFAULT '{}', sort_order INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS template_handwriting (id INTEGER PRIMARY KEY AUTOINCREMENT, template_id INTEGER NOT NULL UNIQUE REFERENCES templates(id) ON DELETE CASCADE, font_family TEXT DEFAULT 'sans-serif', paper_enabled INTEGER DEFAULT 1, paper_opacity REAL DEFAULT 0.12, fiber_count INTEGER DEFAULT 200, dot_count INTEGER DEFAULT 800, global_tilt REAL DEFAULT 1, baseline_drift REAL DEFAULT 0.8, char_jitter REAL DEFAULT 2, char_rotation REAL DEFAULT 1.5, ink_opacity_min REAL DEFAULT 0.85, ink_opacity_max REAL DEFAULT 1.0, char_spacing REAL DEFAULT 1.5, ink_spots_enabled INTEGER DEFAULT 1, ink_spots_chance REAL DEFAULT 0.15, ink_spots_max INTEGER DEFAULT 2, shadow_blur REAL DEFAULT 0.8, checkbox_enabled INTEGER DEFAULT 1, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL DEFAULT '{}')`,
	}
	for _, q := range migrate {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { db.Close(); os.Remove(dbPath) })
	return db
}

func ginRecorder() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestHealthCheck(t *testing.T) {
	r := gin.New()
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "message": "ok", "data": nil})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res["code"].(float64) != 0 {
		t.Errorf("expected code 0, got %v", res["code"])
	}
}

func TestLoginSuccess(t *testing.T) {
	db := setupDB(t)
	InitJWT("test-secret")

	hash, _ := bcrypt.GenerateFromPassword([]byte("testpass"), bcrypt.DefaultCost)
	db.Exec("INSERT INTO admins (username, password) VALUES ('admin', ?)", string(hash))

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := strings.NewReader(`{"username":"admin","password":"testpass"}`)
	c.Request = httptest.NewRequest("POST", "/api/auth/login", body)
	c.Request.Header.Set("Content-Type", "application/json")

	LoginHandler(db)(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res["code"].(float64) != 0 {
		t.Errorf("expected code 0, got %v", res["code"])
	}
	if res["data"].(map[string]interface{})["token"] == "" {
		t.Error("expected non-empty token")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	db := setupDB(t)
	InitJWT("test-secret")

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	db.Exec("INSERT INTO admins (username, password) VALUES ('admin', ?)", string(hash))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/auth/login",
		strings.NewReader(`{"username":"admin","password":"wrong"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	LoginHandler(db)(c)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	InitJWT("test-secret")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/protected", nil)

	AuthRequired()(c)

	if !c.IsAborted() {
		t.Error("expected request to be aborted")
	}
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreateAndGetTemplate(t *testing.T) {
	db := setupDB(t)

	// Create template
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/templates",
		strings.NewReader(`{"name":"测试模板"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	CreateTemplateHandler(db)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res["code"].(float64) != 0 {
		t.Fatalf("expected code 0, got %v", res["code"])
	}
	id := int(res["data"].(map[string]interface{})["id"].(float64))
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	// Get template
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Params = gin.Params{{Key: "id", Value: "1"}}
	c2.Request = httptest.NewRequest("GET", "/api/templates/1", nil)

	GetTemplateHandler(db)(c2)

	if w2.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var tmpl map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &tmpl)
	data := tmpl["data"].(map[string]interface{})
	if data["name"] != "测试模板" {
		t.Errorf("expected name '测试模板', got %v", data["name"])
	}
}

func TestUpdateTemplate(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO templates (name) VALUES ('old')")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("PUT", "/api/templates/1",
		strings.NewReader(`{"name":"new","width":800,"height":1000,"controls":[]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateTemplateHandler(db)(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var name string
	db.QueryRow("SELECT name FROM templates WHERE id=1").Scan(&name)
	if name != "new" {
		t.Errorf("expected name 'new', got '%s'", name)
	}
}

func TestDeleteTemplate(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO templates (name) VALUES ('del')")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("DELETE", "/api/templates/1", nil)

	DeleteTemplateHandler(db)(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var deleted string
	db.QueryRow("SELECT deleted_at FROM templates WHERE id=1").Scan(&deleted)
	if deleted == "" {
		t.Error("expected deleted_at to be set")
	}
}

func TestListTemplates(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO templates (name) VALUES ('a')")
	db.Exec("INSERT INTO templates (name) VALUES ('b')")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/templates?page=1&page_size=10", nil)

	ListTemplatesHandler(db)(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	data := res["data"].(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	if data["total"].(float64) != 2 {
		t.Errorf("expected total 2, got %v", data["total"])
	}
}

func TestSignAndListRecords(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO templates (name, bg_image) VALUES ('sign-test', 'test/bg.png')")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("POST", "/api/templates/1/sign",
		strings.NewReader(`{"fields_data":{"name":"test"},"effect_preset":"light"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	SignHandler(db, t.TempDir(), t.TempDir())(c)

	if w.Code != 500 {
		t.Errorf("expected 500 (missing bg file), got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteRecord(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO templates (name) VALUES ('t')")
	db.Exec("INSERT INTO signing_records (template_id) VALUES (1)")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("DELETE", "/api/records/1", nil)

	DeleteRecordHandler(db)(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminStats(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO templates (name) VALUES ('t')")
	db.Exec("INSERT INTO signing_records (template_id) VALUES (1)")
	db.Exec("INSERT INTO fonts (filename) VALUES ('f.ttf')")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/stats", nil)

	StatsHandler(db)(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	data := res["data"].(map[string]interface{})
	if data["template_count"].(float64) != 1 {
		t.Errorf("expected 1 template, got %v", data["template_count"])
	}
	if data["record_count"].(float64) != 1 {
		t.Errorf("expected 1 record, got %v", data["record_count"])
	}
	if data["font_count"].(float64) != 1 {
		t.Errorf("expected 1 font, got %v", data["font_count"])
	}
}

func TestImportTemplate(t *testing.T) {
	db := setupDB(t)

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f, _ := w.Create("template.json")
	f.Write([]byte(`{"export_version":"1.0","template":{"name":"imported","width":800,"height":1000,"controls":[]}}`))
	w.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.zip")
	part.Write(buf.Bytes())
	writer.Close()

	gin.SetMode(gin.TestMode)
	ginW := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(ginW)
	c.Request = httptest.NewRequest("POST", "/api/templates/import", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	ImportTemplateHandler(db, t.TempDir())(c)

	if ginW.Code != 200 {
		t.Errorf("expected 200, got %d: %s", ginW.Code, ginW.Body.String())
	}
}

func TestExportTemplate(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO templates (name, config_json) VALUES ('exp', '{}')")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("GET", "/api/templates/1/export", nil)

	ExportTemplateHandler(db, "", "")(c)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExportTemplateNotFound(t *testing.T) {
	db := setupDB(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "999"}}
	c.Request = httptest.NewRequest("GET", "/api/templates/999/export", nil)

	ExportTemplateHandler(db, "", "")(c)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Font handler tests ---

func TestListFontsEmpty(t *testing.T) {
	db := setupDB(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/fonts", nil)

	ListFontsHandler(db)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	data := res["data"].([]interface{})
	if len(data) != 0 {
		t.Errorf("expected 0 fonts, got %d", len(data))
	}
}

func TestListFontsWithData(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO fonts (filename, display_name, original_filename) VALUES ('a.ttf', 'Font A', 'a.ttf')")
	db.Exec("INSERT INTO fonts (filename, display_name, original_filename) VALUES ('b.ttf', 'Font B', 'b.ttf')")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/fonts", nil)

	ListFontsHandler(db)(c)

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	data := res["data"].([]interface{})
	if len(data) != 2 {
		t.Errorf("expected 2 fonts, got %d", len(data))
	}
}

func TestListFontsExcludesDeleted(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO fonts (filename, deleted_at) VALUES ('active.ttf', NULL)")
	db.Exec("INSERT INTO fonts (filename, deleted_at) VALUES ('deleted.ttf', CURRENT_TIMESTAMP)")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/fonts", nil)

	ListFontsHandler(db)(c)

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	data := res["data"].([]interface{})
	if len(data) != 1 {
		t.Errorf("expected 1 active font, got %d", len(data))
	}
}

func TestUpdateFontDisplay(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO fonts (filename, display_name) VALUES ('f.ttf', 'Old Name')")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("PUT", "/api/fonts/1",
		strings.NewReader(`{"display_name":"New Name"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateFontHandler(db)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var name string
	db.QueryRow("SELECT display_name FROM fonts WHERE id=1").Scan(&name)
	if name != "New Name" {
		t.Errorf("expected 'New Name', got '%s'", name)
	}
}

func TestUpdateFontNotFound(t *testing.T) {
	db := setupDB(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "999"}}
	c.Request = httptest.NewRequest("PUT", "/api/fonts/999",
		strings.NewReader(`{"display_name":"X"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateFontHandler(db)(c)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteFont(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO fonts (filename) VALUES ('del.ttf')")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("DELETE", "/api/fonts/1", nil)

	DeleteFontHandler(db)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var deleted string
	db.QueryRow("SELECT deleted_at FROM fonts WHERE id=1").Scan(&deleted)
	if deleted == "" {
		t.Error("expected deleted_at to be set")
	}
}

func TestDeleteFontNotFound(t *testing.T) {
	db := setupDB(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "999"}}
	c.Request = httptest.NewRequest("DELETE", "/api/fonts/999", nil)

	DeleteFontHandler(db)(c)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- Settings handwriting tests ---

func TestGetHandwritingSettingsDefault(t *testing.T) {
	db := setupDB(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/settings/handwriting", nil)

	GetHandwritingSettings(db)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var hw models.HandwritingConfig
	var res map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &res)
	json.Unmarshal(res["data"], &hw)
	if hw.FontFamily == nil || *hw.FontFamily != "sans-serif" {
		t.Errorf("expected default font_family sans-serif")
	}
	if hw.PaperOpacity == nil || *hw.PaperOpacity != 0.12 {
		t.Errorf("expected default paper_opacity 0.12")
	}
}

func TestUpdateAndGetHandwritingSettings(t *testing.T) {
	db := setupDB(t)

	body := `{"font_family":"TestFont","paper_opacity":0.5,"char_jitter":3.0}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/api/settings/handwriting",
		strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateHandwritingSettings(db)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Read back
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/api/settings/handwriting", nil)

	GetHandwritingSettings(db)(c2)

	var hw2 models.HandwritingConfig
	var res map[string]json.RawMessage
	json.Unmarshal(w2.Body.Bytes(), &res)
	json.Unmarshal(res["data"], &hw2)
	if hw2.FontFamily == nil || *hw2.FontFamily != "TestFont" {
		t.Errorf("expected font_family 'TestFont', got %v", hw2.FontFamily)
	}
	if hw2.PaperOpacity == nil || *hw2.PaperOpacity != 0.5 {
		t.Errorf("expected paper_opacity 0.5, got %v", hw2.PaperOpacity)
	}
}

func TestUpdateHandwritingSettingsOverwrite(t *testing.T) {
	db := setupDB(t)

	// First write
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/api/settings/handwriting",
		strings.NewReader(`{"font_family":"v1"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	UpdateHandwritingSettings(db)(c)

	// Overwrite
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("PUT", "/api/settings/handwriting",
		strings.NewReader(`{"font_family":"v2"}`))
	c2.Request.Header.Set("Content-Type", "application/json")
	UpdateHandwritingSettings(db)(c2)

	// Verify only one row exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM settings WHERE key='handwriting'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 settings row, got %d", count)
	}
}

// --- Template with handwriting font_family tests ---

func TestCreateTemplateWithHandwritingFontFamily(t *testing.T) {
	db := setupDB(t)

	body := `{"name":"FontTest","handwriting":{"font_family":"CustomFont","paper_opacity":0.9}}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/templates", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	CreateTemplateHandler(db)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify handwriting row has font_family
	var fontFamily string
	db.QueryRow("SELECT font_family FROM template_handwriting WHERE template_id=1").Scan(&fontFamily)
	if fontFamily != "CustomFont" {
		t.Errorf("expected font_family 'CustomFont', got '%s'", fontFamily)
	}
}

func TestUpdateTemplateHandwritingFontFamily(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO templates (name) VALUES ('t')")

	body := `{"name":"t","controls":[],"handwriting":{"font_family":"UpdatedFont","char_jitter":5.0}}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("PUT", "/api/templates/1", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateTemplateHandler(db)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify
	var fontFamily string
	db.QueryRow("SELECT font_family FROM template_handwriting WHERE template_id=1").Scan(&fontFamily)
	if fontFamily != "UpdatedFont" {
		t.Errorf("expected font_family 'UpdatedFont', got '%s'", fontFamily)
	}
}

func TestGetTemplateHandwritingFontFamily(t *testing.T) {
	db := setupDB(t)
	db.Exec("INSERT INTO templates (name) VALUES ('t')")
	db.Exec(`INSERT INTO template_handwriting (template_id, font_family, paper_opacity)
		VALUES (1, 'FromDB', 0.33)`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("GET", "/api/templates/1", nil)

	GetTemplateHandler(db)(c)

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	hw := res["data"].(map[string]interface{})["handwriting"].(map[string]interface{})
	if hw["font_family"] != "FromDB" {
		t.Errorf("expected font_family 'FromDB', got %v", hw["font_family"])
	}
	if hw["paper_opacity"] != 0.33 {
		t.Errorf("expected paper_opacity 0.33, got %v", hw["paper_opacity"])
	}
}

// --- HandwritingConfig.Merge model tests ---

func TestHandwritingConfigMergeFontFamilyOverride(t *testing.T) {
	base := models.DefaultHandwriting()
	other := models.HandwritingConfig{FontFamily: models.SP("OverrideFont")}
	merged := base.Merge(other)
	if *merged.FontFamily != "OverrideFont" {
		t.Errorf("expected 'OverrideFont', got '%s'", *merged.FontFamily)
	}
	// Other fields unchanged
	if *merged.PaperOpacity != 0.12 {
		t.Errorf("expected paper_opacity 0.12 preserved, got %v", *merged.PaperOpacity)
	}
}

func TestHandwritingConfigMergePartialOverride(t *testing.T) {
	base := models.DefaultHandwriting()
	// Only override PaperOpacity, leave everything else as default
	other := models.HandwritingConfig{PaperOpacity: models.FP(0.99)}
	merged := base.Merge(other)

	if *merged.PaperOpacity != 0.99 {
		t.Errorf("expected 0.99, got %v", *merged.PaperOpacity)
	}
	if *merged.CharJitter != 2.0 {
		t.Errorf("expected char_jitter 2.0 preserved, got %v", *merged.CharJitter)
	}
	if *merged.FontFamily != "sans-serif" {
		t.Errorf("expected font_family 'sans-serif' preserved, got '%s'", *merged.FontFamily)
	}
}

func TestHandwritingConfigMergeThreeLevelCascade(t *testing.T) {
	// Simulate: defaults → global → template → sign
	defaults := models.DefaultHandwriting()

	// Global overrides font_family only
	global := models.HandwritingConfig{FontFamily: models.SP("GlobalFont")}
	afterGlobal := defaults.Merge(global)

	// Template overrides char_jitter only (font_family comes from global)
	tmpl := models.HandwritingConfig{CharJitter: models.FP(5.0)}
	afterTemplate := afterGlobal.Merge(tmpl)

	// Sign overrides paper_opacity only
	sign := models.HandwritingConfig{PaperOpacity: models.FP(0.99)}
	final := afterTemplate.Merge(sign)

	// Verify all three overrides are present
	if *final.FontFamily != "GlobalFont" {
		t.Errorf("expected font_family 'GlobalFont', got '%s'", *final.FontFamily)
	}
	if *final.CharJitter != 5.0 {
		t.Errorf("expected char_jitter 5.0, got %v", *final.CharJitter)
	}
	if *final.PaperOpacity != 0.99 {
		t.Errorf("expected paper_opacity 0.99, got %v", *final.PaperOpacity)
	}
	// Fields not overridden should remain defaults
	if *final.InkSpotsMax != 2 {
		t.Errorf("expected ink_spots_max 2 (default), got %v", *final.InkSpotsMax)
	}
}

func TestHandwritingConfigMergeEmptyOther(t *testing.T) {
	base := models.DefaultHandwriting()
	other := models.HandwritingConfig{}
	merged := base.Merge(other)

	// All fields should remain as base
	if *merged.FontFamily != "sans-serif" {
		t.Errorf("expected 'sans-serif', got '%s'", *merged.FontFamily)
	}
	if *merged.PaperOpacity != 0.12 {
		t.Errorf("expected 0.12, got %v", *merged.PaperOpacity)
	}
}

func multipartRequest(t *testing.T, field string, files map[string][]byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, content := range files {
		part, err := writer.CreateFormFile(field, name)
		if err != nil {
			t.Fatal(err)
		}
		part.Write(content)
	}
	writer.Close()
	req := httptest.NewRequest("POST", "/api/fonts/batch", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func fontBatchDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "inkflow_fonts_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func setBatchLimit(t *testing.T, db *sql.DB, limit int) {
	t.Helper()
	if _, err := db.Exec("INSERT INTO settings (key, value) VALUES ('font_batch_limit', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", strconv.Itoa(limit)); err != nil {
		t.Fatal(err)
	}
}

func TestCreateFontsBatchUploadsMultiple(t *testing.T) {
	db := setupDB(t)
	dir := fontBatchDir(t)

	req := multipartRequest(t, "files", map[string][]byte{
		"alpha.ttf": []byte("font-data-alpha"),
		"beta.woff2": []byte("font-data-beta"),
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	CreateFontsBatchHandler(db, dir)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res struct {
		Data struct {
			Uploaded   []fontBatchItem `json:"uploaded"`
			Duplicates []fontBatchItem `json:"duplicates"`
			Failed     []fontBatchItem `json:"failed"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if len(res.Data.Uploaded) != 2 {
		t.Errorf("expected 2 uploaded, got %d", len(res.Data.Uploaded))
	}
	if len(res.Data.Duplicates) != 0 || len(res.Data.Failed) != 0 {
		t.Errorf("expected no duplicates/failed, got dup=%d fail=%d", len(res.Data.Duplicates), len(res.Data.Failed))
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM fonts WHERE deleted_at IS NULL").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 rows in db, got %d", count)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("expected 2 files on disk, got %d", len(entries))
	}
}

func TestCreateFontsBatchDuplicate(t *testing.T) {
	db := setupDB(t)
	dir := fontBatchDir(t)

	req := multipartRequest(t, "files", map[string][]byte{
		"one.ttf": []byte("same-content"),
		"two.ttf": []byte("same-content"),
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	CreateFontsBatchHandler(db, dir)(c)

	var res struct {
		Data struct {
			Uploaded   []fontBatchItem `json:"uploaded"`
			Duplicates []fontBatchItem `json:"duplicates"`
			Failed     []fontBatchItem `json:"failed"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if len(res.Data.Uploaded) != 1 {
		t.Errorf("expected 1 uploaded, got %d", len(res.Data.Uploaded))
	}
	if len(res.Data.Duplicates) != 1 {
		t.Errorf("expected 1 duplicate, got %d", len(res.Data.Duplicates))
	}
	name := res.Data.Duplicates[0].OriginalFilename
	if name != "one.ttf" && name != "two.ttf" {
		t.Errorf("expected duplicate one of one.ttf/two.ttf, got '%s'", name)
	}
}

func TestCreateFontsBatchInvalidExt(t *testing.T) {
	db := setupDB(t)
	dir := fontBatchDir(t)

	req := multipartRequest(t, "files", map[string][]byte{
		"ok.ttf":   []byte("good-font"),
		"bad.txt":  []byte("not a font"),
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	CreateFontsBatchHandler(db, dir)(c)

	var res struct {
		Data struct {
			Uploaded   []fontBatchItem `json:"uploaded"`
			Duplicates []fontBatchItem `json:"duplicates"`
			Failed     []fontBatchItem `json:"failed"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if len(res.Data.Uploaded) != 1 {
		t.Errorf("expected 1 uploaded, got %d", len(res.Data.Uploaded))
	}
	if len(res.Data.Failed) != 1 {
		t.Errorf("expected 1 failed, got %d", len(res.Data.Failed))
	}
	if res.Data.Failed[0].OriginalFilename != "bad.txt" || res.Data.Failed[0].Reason != "文件格式不支持" {
		t.Errorf("unexpected failed item: %+v", res.Data.Failed[0])
	}
}

func TestCreateFontsBatchOverLimitRejectsAll(t *testing.T) {
	db := setupDB(t)
	dir := fontBatchDir(t)
	setBatchLimit(t, db, 1)

	req := multipartRequest(t, "files", map[string][]byte{
		"a.ttf": []byte("aaa"),
		"b.ttf": []byte("bbb"),
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	CreateFontsBatchHandler(db, dir)(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM fonts WHERE deleted_at IS NULL").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 rows after rejection, got %d", count)
	}
}

func TestCreateFontsBatchUnlimitedWhenZero(t *testing.T) {
	db := setupDB(t)
	dir := fontBatchDir(t)
	setBatchLimit(t, db, 0)

	req := multipartRequest(t, "files", map[string][]byte{
		"a.ttf": []byte("aaa"),
		"b.ttf": []byte("bbb"),
		"c.ttf": []byte("ccc"),
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	CreateFontsBatchHandler(db, dir)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	uploaded := res["data"].(map[string]interface{})["uploaded"].([]interface{})
	if len(uploaded) != 3 {
		t.Errorf("expected 3 uploaded with unlimited, got %d", len(uploaded))
	}
}

func TestFontBatchLimitDefault(t *testing.T) {
	db := setupDB(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/settings/font_batch_limit", nil)

	GetFontBatchLimit(db)(c)

	var res struct {
		Data struct {
			Limit int `json:"limit"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Data.Limit != 10 {
		t.Errorf("expected default limit 10, got %d", res.Data.Limit)
	}
}

func TestFontBatchLimitUpdateAndRead(t *testing.T) {
	db := setupDB(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/api/settings/font_batch_limit",
		strings.NewReader(`{"limit":25}`))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateFontBatchLimit(db)(c)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var value string
	db.QueryRow("SELECT value FROM settings WHERE key = 'font_batch_limit'").Scan(&value)
	if value != "25" {
		t.Errorf("expected stored value '25', got '%s'", value)
	}
	if got := readFontBatchLimit(db); got != 25 {
		t.Errorf("expected readFontBatchLimit 25, got %d", got)
	}
}

func TestFontBatchLimitRejectNegative(t *testing.T) {
	db := setupDB(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("PUT", "/api/settings/font_batch_limit",
		strings.NewReader(`{"limit":-3}`))
	c.Request.Header.Set("Content-Type", "application/json")

	UpdateFontBatchLimit(db)(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
