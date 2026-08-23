package server

import (
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/crypto/bcrypt"
	"inkflow-go/internal/database"
)

func InitAdmin(dbPath, password string) error {
	if err := database.Init(dbPath); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = database.DB.Exec("INSERT OR REPLACE INTO admins (username, password) VALUES ('admin', ?)", string(hash))
	if err != nil {
		return err
	}
	fmt.Println("管理员账号初始化成功")
	return nil
}

func initAdminAuto(dbPath string) {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	if count > 0 {
		return
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin"
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	database.DB.Exec("INSERT INTO admins (username, password) VALUES ('admin', ?)", string(hash))
	slog.Info("initial admin created", "username", "admin")
}
