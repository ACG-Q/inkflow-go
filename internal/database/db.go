package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	DB.SetMaxOpenConns(4)
	DB.SetMaxIdleConns(2)

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA temp_store=MEMORY",
	}
	for _, p := range pragmas {
		if _, err := DB.Exec(p); err != nil {
			slog.Warn("pragma failed", "sql", p, "error", err)
		}
	}

	return migrate()
}

func migrate() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL DEFAULT '未命名模板',
			bg_image TEXT,
			width INTEGER DEFAULT 800,
			height INTEGER DEFAULT 1000,
			config_json TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS signing_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			template_id INTEGER NOT NULL REFERENCES templates(id),
			fields_data TEXT,
			image_url TEXT,
			ip_address TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS fonts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			display_name TEXT,
			original_filename TEXT,
			file_hash TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME DEFAULT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE DEFAULT 'admin',
			password TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_records_template_id ON signing_records(template_id)`,
		`CREATE INDEX IF NOT EXISTS idx_records_created_at ON signing_records(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_deleted_at ON templates(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_records_deleted_at ON signing_records(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_fonts_deleted_at ON fonts(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_fonts_file_hash ON fonts(file_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_name ON templates(name)`,

		`CREATE TABLE IF NOT EXISTS template_controls (
			id             TEXT PRIMARY KEY,
			template_id    INTEGER NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
			label          TEXT NOT NULL DEFAULT '',
			type           TEXT NOT NULL DEFAULT 'textbox',
			x              REAL DEFAULT 0,
			y              REAL DEFAULT 0,
			width          REAL DEFAULT 200,
			height         REAL DEFAULT 40,
			font_size      INTEGER DEFAULT 20,
			font_family    TEXT DEFAULT 'sans-serif',
			required       INTEGER DEFAULT 0,
			preview_text   TEXT DEFAULT '',
			check_size     INTEGER DEFAULT 24,
			sort_order     INTEGER DEFAULT 0,
			created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS template_rules (
			id             TEXT PRIMARY KEY,
			template_id    INTEGER NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
			type           TEXT NOT NULL,
			name           TEXT DEFAULT '',
			target         TEXT DEFAULT '',
			config_json    TEXT NOT NULL DEFAULT '{}',
			sort_order     INTEGER DEFAULT 0,
			created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS template_handwriting (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			template_id       INTEGER NOT NULL UNIQUE REFERENCES templates(id) ON DELETE CASCADE,
			paper_enabled     INTEGER DEFAULT 1,
			paper_opacity     REAL DEFAULT 0.12,
			fiber_count       INTEGER DEFAULT 200,
			dot_count         INTEGER DEFAULT 800,
			global_tilt       REAL DEFAULT 1,
			baseline_drift    REAL DEFAULT 0.8,
			char_jitter       REAL DEFAULT 2,
			char_rotation     REAL DEFAULT 1.5,
			ink_opacity_min   REAL DEFAULT 0.85,
			ink_opacity_max   REAL DEFAULT 1.0,
			char_spacing      REAL DEFAULT 1.5,
			ink_spots_enabled INTEGER DEFAULT 1,
			ink_spots_chance  REAL DEFAULT 0.15,
			ink_spots_max     INTEGER DEFAULT 2,
			shadow_blur       REAL DEFAULT 0.8,
			checkbox_enabled  INTEGER DEFAULT 1,
			created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at        DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tc_template_id ON template_controls(template_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tr_template_id ON template_rules(template_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tr_type ON template_rules(type)`,
	}
	for _, t := range tables {
		if _, err := DB.Exec(t); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	// Guard ALTER TABLE for upgrades: only add font_family if it doesn't already exist
	var colCount int
	DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('template_handwriting') WHERE name='font_family'").Scan(&colCount)
	if colCount == 0 {
		if _, err := DB.Exec("ALTER TABLE template_handwriting ADD COLUMN font_family TEXT DEFAULT 'sans-serif'"); err != nil {
			return fmt.Errorf("migrate alter font_family: %w", err)
		}
	}
	return nil
}
