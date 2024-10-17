package login

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golang-backend/database"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// User struct สำหรับจัดการข้อมูลผู้ใช้
type User struct {
	ID        int     `json:"id"`
	Username  string  `json:"username"`
	FirstName string  `json:"firstname"`
	LastName  string  `json:"lastname"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	Role      string  `json:"role"`
	Password  string  `json:"password,omitempty"` // Exclude password from being output in JSON
	CreatedAt string  `json:"created_at"`
	TeamId    *int    `json:"team_id"`
	TeamName  *string `json:"team_name"`
}

var jwtKey = []byte("c52d0fb7e13a8db018af68aff4c2684162cd9b641fe4d22180eae5d7499a329b074ddebcc254a58539e5c5eea9a0d8fb4c7c1123d2996bd219b4738ae5ce91f8e9bfdfa0d851f7ace899ba4c292c3ebfd0da11063531f0b8805748409df7324367053f71c62be461107dda56a81600a8db34762efc458bf25bf7c31f5e39e411fc247dd9926ee6bd40b56c6fedc4b602a2a67941ddbd9bd739f7573bb23c099466d1b7c8a219721af7122ab9e5fa3207c490ce2476e784b1d719b2ff6c8d371a39ccf60a2c7a974d3b7a25fe1d61aeb4912a005c6a58c561110aa34fe3e1ed2242ee53661ae57bda286d6d365a0ef14e5c2a59e67f4db6148d495042fe6b634f") // อย่าลืมเปลี่ยนเป็นคีย์ที่ปลอดภัย

// โครงสร้างของ JWT Claims
type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func CreateToken(username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // ตั้งเวลาหมดอายุของ JWT

	claims := &Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey) // เซ็นชื่อด้วยคีย์ลับ
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// ฟังก์ชันสำหรับตรวจสอบรหัสผ่าน
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ฟังก์ชันสำหรับ login
func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var loginData struct {
		Identifier string `json:"identifier"` // อาจจะเป็น username หรือ email
		Password   string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// ตรวจสอบว่าค่าที่ได้รับไม่เป็นค่าว่าง
	if loginData.Identifier == "" || loginData.Password == "" {
		http.Error(w, "Username/email and password are required", http.StatusBadRequest)
		return
	}

	var user User
	query := `
		SELECT id, username, firstname, lastname, email, phone, role, password, created_at 
		FROM users 
		WHERE username = ? OR email = ?
	`
	row := database.DB.QueryRow(query, loginData.Identifier, loginData.Identifier)

	if err := row.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.Email, &user.Phone, &user.Role, &user.Password, &user.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			// บันทึกเหตุการณ์การเข้าสู่ระบบที่ล้มเหลว
			fmt.Println("Invalid login attempt for identifier:", loginData.Identifier)
			http.Error(w, "Invalid username or email", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Query error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !CheckPasswordHash(loginData.Password, user.Password) {
		// บันทึกเหตุการณ์การเข้าสู่ระบบที่ล้มเหลว
		fmt.Println("Invalid password attempt for identifier:", loginData.Identifier)
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	// สร้าง JWT token
	token, err := CreateToken(user.Username)
	if err != nil {
		http.Error(w, "Could not create token", http.StatusInternalServerError)
		return
	}

	// ส่ง JWT token กลับไปยังผู้ใช้
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Login successful",
		"user":    user,
		"token":   token, // ส่ง token กลับไป
	})
}
