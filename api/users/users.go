package user

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golang-backend/database"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

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

// GetUsers godoc
// @Summary Get all users
// @Description Get details of all users
// @Tags users
// @Accept  json
// @Produce  json
// @Success 200 {array} User
// @Router /users [get]
func GetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ปรับปรุง SQL Query เพื่อทำการ JOIN ข้อมูลจาก teams
	query := `
		SELECT u.id, u.username, u.firstname, u.lastname, u.email, u.phone, u.role, u.created_at, t.team_id, t.team_name
		FROM users u
		LEFT JOIN teams t ON u.team_id = t.team_id
	`

	stmt, err := database.DB.Prepare(query)
	if err != nil {
		http.Error(w, "Database preparation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Execute the query
	rows, err := stmt.Query()
	if err != nil {
		http.Error(w, "Query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		// ดึงข้อมูลจากทั้งตาราง users และ teams
		if err := rows.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.Email, &user.Phone, &user.Role, &user.CreatedAt, &user.TeamId, &user.TeamName); err != nil {
			http.Error(w, "Error scanning row: "+err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"users": users})
}

func GetUsersByTeam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ดึง team_id จาก params
	params := mux.Vars(r)
	teamID := params["team_id"]

	// ตรวจสอบว่า team_id ถูกส่งมาใน URL หรือไม่
	if teamID == "" {
		http.Error(w, "Team ID is required", http.StatusBadRequest)
		return
	}

	// Query เพื่อดึงข้อมูลผู้ใช้จาก users และ team_name จาก teams โดยกรองตาม team_id
	query := `
			SELECT users.id, users.username, users.firstname, users.lastname, users.email, users.phone, users.role, users.created_at, users.team_id, teams.team_name
			FROM users
			LEFT JOIN teams ON users.team_id = teams.team_id
			WHERE users.team_id = ?
	`

	rows, err := database.DB.Query(query, teamID)
	if err != nil {
		http.Error(w, "Error executing query: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.Email, &user.Phone, &user.Role, &user.CreatedAt, &user.TeamId, &user.TeamName); err != nil {
			http.Error(w, "Error scanning row: "+err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Row error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ส่งข้อมูลผู้ใช้ที่กรองตาม team_id กลับไปในรูปแบบ JSON
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"users": users})
}

func GetUserByID(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from the URL using mux.Vars
	vars := mux.Vars(r)
	id := vars["id"]

	// Query the user by ID from the database
	stmt, err := database.DB.Prepare("SELECT id, username, firstname, lastname, email, phone, role, created_at FROM users WHERE id = ?")
	if err != nil {
		http.Error(w, "Error preparing query: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Query the user by ID from the database
	var user User
	err = stmt.QueryRow(id).Scan(
		&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.Email, &user.Phone, &user.Role, &user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error fetching user: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return the user data in the desired JSON format
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"user": user})
}

// CreateUser godoc
// @Summary Create a new user
// @Description Create a new user and hash the password before saving to the database
// @Tags users
// @Accept  json
// @Produce  json
// @Param user body User true "User data"
// @Success 201 {object} map[string]string{"message": "User created successfully"}
// @Failure 400 {object} map[string]string{"error": "Invalid input"}
// @Failure 500 {object} map[string]string{"error": "Internal Server Error"}
// @Router /users [post]
func CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// ตรวจสอบข้อมูลที่จำเป็น
	if user.Username == "" || user.Password == "" || user.Email == "" {
		http.Error(w, "Username, password, and email are required", http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	// ใช้ Prepared Statement เพื่อป้องกัน SQL Injection
	stmt, err := database.DB.Prepare(`
		INSERT INTO users (username, password, firstname, lastname, email, phone, role, team_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW())
	`)
	if err != nil {
		http.Error(w, "Error preparing statement: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Execute statement พร้อมกับค่าที่ผ่านการกรอง
	_, err = stmt.Exec(user.Username, hashedPassword, user.FirstName, user.LastName, user.Email, user.Phone, user.Role, user.TeamId)
	if err != nil {
		http.Error(w, "Error creating user: "+err.Error(), http.StatusInternalServerError)
		fmt.Println("Database exec error:", err)
		return
	}

	// กำหนดเวลา created_at ในรูปแบบที่ต้องการ
	user.CreatedAt = time.Now().Format("2006-01-02 15:04:05")

	// ส่งข้อมูลผู้ใช้กลับในรูปแบบ JSON
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func DeleteUserByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Extract the user ID from the URL
	vars := mux.Vars(r)
	id := vars["id"]

	// ใช้ Prepared Statement เพื่อป้องกัน SQL Injection
	stmt, err := database.DB.Prepare("DELETE FROM users WHERE id = ?")
	if err != nil {
		http.Error(w, "Error preparing statement: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Execute the delete query with the user ID as a parameter
	result, err := stmt.Exec(id)
	if err != nil {
		http.Error(w, "Error deleting user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected (if the user was deleted)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error checking rows affected: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// If no rows were affected, the user was not found
	if rowsAffected == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Respond with a success message
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User deleted successfully"})
}

func PatchUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get the user ID from the request URL (assuming user ID is passed as a URL parameter)
	vars := mux.Vars(r)
	userId := vars["id"]

	var userUpdates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&userUpdates); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Start building the SQL query dynamically based on the fields that are updated
	query := "UPDATE users SET"
	params := []interface{}{}
	setClauses := []string{}

	// Check for fields to update and add them to the query
	if userName, ok := userUpdates["username"]; ok {
		setClauses = append(setClauses, "username = ?")
		params = append(params, userName)
	}
	if firstName, ok := userUpdates["firstname"]; ok {
		setClauses = append(setClauses, "firstname = ?")
		params = append(params, firstName)
	}
	if lastName, ok := userUpdates["lastname"]; ok {
		setClauses = append(setClauses, "lastname = ?")
		params = append(params, lastName)
	}
	if email, ok := userUpdates["email"]; ok {
		setClauses = append(setClauses, "email = ?")
		params = append(params, email)
	}
	if phone, ok := userUpdates["phone"]; ok {
		setClauses = append(setClauses, "phone = ?")
		params = append(params, phone)
	}
	if role, ok := userUpdates["role"]; ok {
		setClauses = append(setClauses, "role = ?")
		params = append(params, role)
	}
	if password, ok := userUpdates["password"]; ok {
		// Hash the new password before updating it
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password.(string)), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}
		setClauses = append(setClauses, "password = ?")
		params = append(params, hashedPassword)
	}
	if teamID, ok := userUpdates["team_id"]; ok {
		setClauses = append(setClauses, "team_id = ?")
		params = append(params, teamID)
	}

	if len(setClauses) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	// Join the SET clauses and complete the SQL query
	query += " " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	params = append(params, userId)

	// Use prepared statement for security
	stmt, err := database.DB.Prepare(query)
	if err != nil {
		http.Error(w, "Error preparing statement: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Execute the query with the user ID as a parameter
	_, err = stmt.Exec(params...)
	if err != nil {
		http.Error(w, "Error updating user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User updated successfully"})
}
