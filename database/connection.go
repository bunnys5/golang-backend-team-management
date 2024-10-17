package database

import (
    "database/sql"
    "fmt"
    "log"

    _ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Connect() {
    // กำหนด DSN (Data Source Name) สำหรับการเชื่อมต่อ MySQL
    dsn := "root:@tcp(127.0.0.1:3306)/golang_project"
    
    // สร้างการเชื่อมต่อฐานข้อมูล
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // ทดสอบการเชื่อมต่อว่าทำงานได้ถูกต้องหรือไม่
    err = db.Ping()
    if err != nil {
        log.Fatal("Failed to ping database:", err)
    }

    // ถ้าเชื่อมต่อสำเร็จ เก็บไว้ในตัวแปร DB
    DB = db
    fmt.Println("Database connection established")
}
