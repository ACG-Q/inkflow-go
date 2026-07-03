package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthRequired 验证 JWT
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
			tokenStr = tokenStr[7:]
		}
		if tokenStr == "" {
			Error(c, http.StatusUnauthorized, 40101, "未登录或 token 过期")
			c.Abort()
			return
		}
		claims, err := ParseToken(tokenStr)
		if err != nil {
			Error(c, http.StatusUnauthorized, 40101, "未登录或 token 过期")
			c.Abort()
			return
		}
		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

type rateLimiter struct {
	mu    sync.Mutex
	visits map[string]*visit
}

type visit struct {
	count    int
	expireAt time.Time
}

func RateLimit(maxPerMinute int) gin.HandlerFunc {
	rl := &rateLimiter{visits: make(map[string]*visit)}
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.mu.Lock()
			for ip, v := range rl.visits {
				if time.Now().After(v.expireAt) {
					delete(rl.visits, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return func(c *gin.Context) {
		if maxPerMinute <= 0 {
			c.Next()
			return
		}
		ip := c.ClientIP()
		rl.mu.Lock()
		v, ok := rl.visits[ip]
		if !ok || time.Now().After(v.expireAt) {
			rl.visits[ip] = &visit{count: 1, expireAt: time.Now().Add(time.Minute)}
			rl.mu.Unlock()
			c.Next()
			return
		}
		v.count++
		if v.count > maxPerMinute {
			rl.mu.Unlock()
			Error(c, http.StatusTooManyRequests, 40005, "请求过于频繁")
			c.Abort()
			return
		}
		rl.mu.Unlock()
		c.Next()
	}
}