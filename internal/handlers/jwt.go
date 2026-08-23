package handlers

import (
	"crypto/rand"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func InitJWT(secret string) {
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic("failed to generate JWT secret: " + err.Error())
		}
		jwtSecret = b
		slog.Warn("JWT_SECRET not set, using auto-generated secret (tokens will not survive restart)")
	} else {
		jwtSecret = []byte(secret)
	}
}

type Claims struct {
	AdminID  int64  `json:"admin_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateToken(adminID int64, username string) (string, error) {
	claims := Claims{
		AdminID:  adminID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}
