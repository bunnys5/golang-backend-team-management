package teams

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"golang-backend/database"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// User struct สำหรับจัดการข้อมูลผู้ใช้
type Teams struct {
	ID        int    `json:"team_id"`
	TeamName  string `json:"team_name"`
	CreatedAt string `json:"created_at"`
}

// ฟังก์ชันสำหรับ hash รหัสผ่าน

// GetUsers godoc
// @Summary Get all users
// @Description Get details of all users
// @Tags users
// @Accept  json
// @Produce  json
// @Success 200 {array} User
// @Router /users [get]
func GetTeams(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ปรับปรุง SQL Query เพื่อทำการ JOIN ข้อมูลจาก teams
	query := `
		SELECT team_id, team_name, created_at from teams
	`

	stmt, err := database.DB.Prepare(query)
	if err != nil {
		http.Error(w, "Query preparation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query() // ใช้ stmt.Query() แทน database.DB.Query()
	if err != nil {
		http.Error(w, "Query execution error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var teams []Teams
	for rows.Next() {
		var team Teams
		// ดึงข้อมูลจากทั้งตาราง users และ teams
		if err := rows.Scan(&team.ID, &team.TeamName, &team.CreatedAt); err != nil {
			http.Error(w, "Error scanning row: "+err.Error(), http.StatusInternalServerError)
			return
		}
		teams = append(teams, team)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"teams": teams})
}

func CreateTeam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var team Teams
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if team.TeamName == "" {
		http.Error(w, "Team Name is required", http.StatusBadRequest)
		return
	}

	// สร้าง Prepared Statement สำหรับการแทรกข้อมูล
	stmt, err := database.DB.Prepare(`INSERT INTO teams (team_name, created_at) VALUES (?, NOW())`)
	if err != nil {
		http.Error(w, "Query preparation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close() // ปิด Prepared Statement หลังการใช้งาน

	// Execute the query with the team name
	_, err = stmt.Exec(team.TeamName)
	if err != nil {
		http.Error(w, "Error creating teams: "+err.Error(), http.StatusInternalServerError)
		fmt.Println("Database exec error:", err)
		return
	}

	// Retrieve the created_at value to include in the response
	team.CreatedAt = time.Now().Format("2006-01-02 15:04:05") // Example format for MySQL-compatible databases

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(team)
}

func GetTeamById(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from the URL using mux.Vars
	vars := mux.Vars(r)
	id := vars["id"]

	// Validate the ID format (optional but recommended)
	if id == "" {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	// Prepare the SQL query to prevent SQL injection
	query := "SELECT team_id, team_name, created_at FROM teams WHERE team_id = ?"
	stmt, err := database.DB.Prepare(query)
	if err != nil {
		http.Error(w, "Query preparation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close() // Ensure the statement is closed after execution

	// Query the team by ID from the database
	var team Teams
	err = stmt.QueryRow(id).Scan(&team.ID, &team.TeamName, &team.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Team not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error fetching team: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return the team data in the desired JSON format
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"team": team})
}

func DeleteTeamById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract the team ID from the URL
	vars := mux.Vars(r)
	teamID := vars["team_id"]

	// Validate the team ID format (optional but recommended)
	if teamID == "" {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	// Prepare the delete query to prevent SQL injection
	query := "DELETE FROM teams WHERE team_id = ?"
	stmt, err := database.DB.Prepare(query)
	if err != nil {
		http.Error(w, "Query preparation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close() // Ensure the statement is closed after execution

	// Execute the delete query
	result, err := stmt.Exec(teamID)
	if err != nil {
		http.Error(w, "Error deleting team: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected (if the team was deleted)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Error checking rows affected: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// If no rows were affected, the team was not found
	if rowsAffected == 0 {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	// Respond with a success message
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Team deleted successfully"})
}

func PatchTeam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get the team ID from the request URL (assuming team ID is passed as a URL parameter)
	vars := mux.Vars(r)
	teamId := vars["team_id"]

	var teamUpdate map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&teamUpdate); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Start building the SQL query dynamically based on the fields that are updated
	query := "UPDATE teams SET"
	params := []interface{}{}
	setClauses := []string{}

	// Check for fields to update and add them to the query
	if teamName, ok := teamUpdate["team_name"]; ok {
		setClauses = append(setClauses, "team_name = ?")
		params = append(params, teamName)
	}

	if len(setClauses) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	// Join the SET clauses and complete the SQL query
	query += " " + strings.Join(setClauses, ", ") + " WHERE team_id = ?"
	params = append(params, teamId)

	// Prepare the SQL statement
	stmt, err := database.DB.Prepare(query)
	if err != nil {
		http.Error(w, "Query preparation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close() // Ensure the statement is closed after execution

	// Execute the query
	_, err = stmt.Exec(params...)
	if err != nil {
		http.Error(w, "Error updating teams: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Team updated successfully"})
}
