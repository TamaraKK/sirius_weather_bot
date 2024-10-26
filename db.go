package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

func initDatabase() {
	errr := godotenv.Load()
	if errr != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connStr := "user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		chat_id BIGINT UNIQUE NOT NULL,
		city VARCHAR(255) NOT NULL,
		frequency VARCHAR(255) NOT NULL
		);
	`)
	if err != nil {
		log.Fatal("Error creating table:", err)
	}
}

func getCityByChatID(chatID int64) (string, error) {
	var city string
	err := db.QueryRow("SELECT city FROM users WHERE chat_id = $1", chatID).Scan(&city)
	return city, err
}

func updateUser(chatID int64, city string, frequency string) error {
	_, err := db.Exec("INSERT INTO users (chat_id, city, frequency) VALUES ($1, $2, $3) ON CONFLICT (chat_id) DO UPDATE SET city=$2, frequency=$3", chatID, city, frequency)
	return err
}

func updateFrequency(chatID int64, frequency string) error {
	_, err := db.Exec("UPDATE users SET frequency = $1 WHERE chat_id = $2", frequency, chatID)
	return err
}
