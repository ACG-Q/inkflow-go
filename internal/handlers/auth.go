package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func LoginHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			Error(c, http.StatusBadRequest, 40001, "参数校验失败")
			return
		}
		var id int64
		var hash string
		err := db.QueryRow("SELECT id, password FROM admins WHERE username = ?", req.Username).Scan(&id, &hash)
		if err != nil {
			Error(c, http.StatusUnauthorized, 40102, "用户名或密码错误")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
			Error(c, http.StatusUnauthorized, 40102, "用户名或密码错误")
			return
		}
		token, err := GenerateToken(id, req.Username)
		if err != nil {
			Error(c, http.StatusInternalServerError, 50001, "数据库错误")
			return
		}
		Success(c, gin.H{"token": token})
	}
}

func MeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, _ := c.Get("username")
		Success(c, gin.H{"username": username})
	}
}
