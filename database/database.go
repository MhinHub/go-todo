// package database

// import (
// 	"database/sql"
// 	_ "github.com/lib/pq"
// )

// func InitDb() *sql.DB {
// 	dsn := "postgres://postgres:123@localhost:5432/todos?sslmode=disable"
// 	sqlDB, err := sql.Open("postgres", dsn)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return sqlDB
// }

package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func InitDb() *sql.DB {
	// Memuat variabel dari file .env
	// Tanda "_" digunakan jika kita tidak butuh menangani error secara spesifik,
	// tapi lebih baik menanganinya.
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Could not find .env file, using environment variables from OS")
	}

	// Membaca setiap variabel dari environment
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	// Membangun Data Source Name (DSN) dari environment variables
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
		dbSSLMode,
	)

	// sql.Open tetap sama
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		// Sebaiknya gunakan log.Fatal agar lebih informatif daripada panic()
		log.Fatalf("Could not connect to the database: %v", err)
	}

	return sqlDB
}
