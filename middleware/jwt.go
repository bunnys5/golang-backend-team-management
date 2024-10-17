package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4" // ใช้ไลบรารี JWT
)

// JWTMiddleware ตรวจสอบ JWT token
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// รับค่า Authorization header
		tokenString := r.Header.Get("Authorization")
		tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))

		if tokenString == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// ตรวจสอบ token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// ตรวจสอบว่า algorithm ที่ใช้เป็น HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte("c52d0fb7e13a8db018af68aff4c2684162cd9b641fe4d22180eae5d7499a329b074ddebcc254a58539e5c5eea9a0d8fb4c7c1123d2996bd219b4738ae5ce91f8e9bfdfa0d851f7ace899ba4c292c3ebfd0da11063531f0b8805748409df7324367053f71c62be461107dda56a81600a8db34762efc458bf25bf7c31f5e39e411fc247dd9926ee6bd40b56c6fedc4b602a2a67941ddbd9bd739f7573bb23c099466d1b7c8a219721af7122ab9e5fa3207c490ce2476e784b1d719b2ff6c8d371a39ccf60a2c7a974d3b7a25fe1d61aeb4912a005c6a58c561110aa34fe3e1ed2242ee53661ae57bda286d6d365a0ef14e5c2a59e67f4db6148d495042fe6b634f"), nil // ใช้ secret key ของคุณ
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// หาก token ถูกต้อง ให้ไปยัง handler ถัดไป
		next.ServeHTTP(w, r)
	})
}
