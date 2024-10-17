package main

import (
	"fmt"
	"golang-backend/api/login"
	"golang-backend/api/teams"
	user "golang-backend/api/users"
	"golang-backend/database"
	_ "golang-backend/docs"
	"golang-backend/middleware" // นำเข้า middleware
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// เชื่อมต่อกับฐานข้อมูล
	database.Connect()

	// ตั้งค่า CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// สร้าง router
	router := mux.NewRouter()

	// เส้นทางหลัก
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Welcome to the User Management API!")
	}).Methods("GET")

	// เส้นทางจัดการผู้ใช้ (ไม่มีการตรวจสอบ JWT)
	router.HandleFunc("/login", login.Login).Methods("POST")

	// ใช้ middleware JWT สำหรับเส้นทางที่ต้องการ
	api := router.PathPrefix("/api").Subrouter() // ใช้ subrouter สำหรับ API
	api.Use(middleware.JWTMiddleware)            // ใช้ middleware

	api.HandleFunc("/users", user.GetUsers).Methods("GET")
	api.HandleFunc("/users/{id}", user.GetUserByID).Methods("GET")
	api.HandleFunc("/users/team/{team_id}", user.GetUsersByTeam).Methods("GET")
	api.HandleFunc("/users", user.CreateUser).Methods("POST")
	api.HandleFunc("/users/{id}", user.DeleteUserByID).Methods("DELETE")
	api.HandleFunc("/users/{id}", user.PatchUser).Methods("PATCH")
	api.HandleFunc("/teams", teams.GetTeams).Methods("GET")
	api.HandleFunc("/teams/{id}", teams.GetTeamById).Methods("GET")
	api.HandleFunc("/teams", teams.CreateTeam).Methods("POST")
	api.HandleFunc("/teams/{team_id}", teams.PatchTeam).Methods("PATCH")
	api.HandleFunc("/teams/{team_id}", teams.DeleteTeamById).Methods("DELETE")

	// Wrap the router with the CORS handler
	handler := c.Handler(router)

	// Run server on port 8080
	fmt.Println("Server is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
