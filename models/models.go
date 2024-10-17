package models

// User struct to represent a user in the users table
type User struct {
    ID        int    `json:"id"`
    Username  string `json:"username"`
    FirstName string `json:"first_name"`
    LastName  string `json:"last_name"`
    Email     string `json:"email"`
    Phone     string `json:"phone"`
    Password  string `json:"password"`
}