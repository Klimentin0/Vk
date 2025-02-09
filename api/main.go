package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

type PingResult struct {
	ContainerID   string  `json:"container_id"`
	ContainerName string  `json:"container_name"`
	PingDuration  float64 `json:"ping_duration"`
	Status        string  `json:"status"`
}

// Инициализируемв в API жц базу данных
func initDB() {
	var err error
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)
	log.Println("Connecting to DB with:", connStr)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Создаём таблицу, проверяем нет ли уже существующей.
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS ping_results (
        id SERIAL PRIMARY KEY,
        container_id TEXT,
        container_name TEXT,
        ping_duration DOUBLE PRECISION,
        status TEXT,
        timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Database initialized successfully.")
}

// Отправлем результат пинга в бд
func savePingResult(result PingResult) {
	_, err := db.Exec(
		"INSERT INTO ping_results (container_id, container_name, ping_duration, status) VALUES ($1, $2, $3, $4)",
		result.ContainerID, result.ContainerName, result.PingDuration, result.Status,
	)
	if err != nil {
		log.Printf("Error saving ping result for container %s: %v", result.ContainerID, err)
	} else {
		log.Printf("Saved ping result for container %s", result.ContainerID)
	}
}

// Принимаем POST от Пингера
func handlePingResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Декодим JSON
	var result PingResult
	err := json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	savePingResult(result)

	fmt.Fprintf(w, "Ping result saved: %s - %s\n", result.ContainerID, result.Status)
}

// Для запроса от фронта
func getPingResults(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT container_id, container_name, ping_duration, status, timestamp FROM ping_results ORDER BY timestamp DESC")
	if err != nil {
		http.Error(w, "Error fetching ping results", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Парсим в слайс []PingResult ответ
	var results []PingResult
	for rows.Next() {
		var result PingResult
		var timestamp time.Time
		err := rows.Scan(&result.ContainerID, &result.ContainerName, &result.PingDuration, &result.Status, &timestamp)
		if err != nil {
			http.Error(w, "Error scanning ping results", http.StatusInternalServerError)
			return
		}
		results = append(results, result)
	}

	// Возвращаем фронту JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func main() {
	initDB()
	defer db.Close()
	//Раутинг
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Placeholder")
	})
	http.HandleFunc("/ping-results", handlePingResults)
	http.HandleFunc("/ping-results/all", getPingResults)
	//Старт сервера
	fmt.Println("API is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Просто команда для линукса чтобы быстро проверять постгрес
// psql -h localhost -p 5432 -U postgres -d status-check-db
