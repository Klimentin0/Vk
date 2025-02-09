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
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"container_name"`
	PingDuration  float64   `json:"ping_duration"`
	Status        string    `json:"status"`
	IPAddress     string    `json:"ip_address"`
	Timestamp     time.Time `json:"timestamp"`
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
	// НАДО ДАТЬ ВРЕМЯ ПРИ ПЕРВОМ ЗАПУСКЕ инициализироваться дб-шке
	maxRetries := 10
	retryDelay := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Attempt %d: Failed to open database connection: %v", i+1, err)
		} else {
			// Тестируем коннект
			err = db.Ping()
			if err == nil {
				log.Println("Database connection established successfully.")
				break
			}
			log.Printf("Attempt %d: Database ping failed: %v", i+1, err)
		}

		if i == maxRetries-1 {
			log.Fatalf("Failed to connect to the database after %d attempts", maxRetries)
		}

		log.Printf("Retrying in %v...", retryDelay)
		time.Sleep(retryDelay)
	}

	// Создаём таблицу, проверяем нет ли уже существующей.
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS ping_results (
        id SERIAL PRIMARY KEY,
        container_id TEXT,
        container_name TEXT,
        ping_duration DOUBLE PRECISION,
        status TEXT,
		ip_address TEXT,
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
		"INSERT INTO ping_results (container_id, container_name, ping_duration, status, ip_address) VALUES ($1, $2, $3, $4, $5)",
		result.ContainerID, result.ContainerName, result.PingDuration, result.Status, result.IPAddress,
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
	rows, err := db.Query("SELECT container_id, container_name, ping_duration, status, ip_address, timestamp FROM ping_results ORDER BY timestamp DESC")
	if err != nil {
		http.Error(w, "Error fetching ping results", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Парсим в слайс []PingResult ответ
	var results []PingResult
	for rows.Next() {
		var result PingResult
		err := rows.Scan(&result.ContainerID, &result.ContainerName, &result.PingDuration, &result.Status, &result.IPAddress, &result.Timestamp)
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

func getLatestUPPerContainer(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
        SELECT DISTINCT ON (container_name) container_id, container_name, ping_duration, status, ip_address, timestamp
        FROM ping_results
        WHERE status = 'UP'
        ORDER BY container_name, timestamp DESC
    `)
	if err != nil {
		http.Error(w, "Error fetching latest UP results per container", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []PingResult
	for rows.Next() {
		var result PingResult
		err := rows.Scan(&result.ContainerID, &result.ContainerName, &result.PingDuration, &result.Status, &result.IPAddress, &result.Timestamp)
		if err != nil {
			http.Error(w, "Error scanning latest UP results per container", http.StatusInternalServerError)
			return
		}
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// Нужно разобраться с CORS
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			return // preflight requests
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	initDB()
	defer db.Close()

	mux := http.NewServeMux()

	//Раутинг
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Placeholder")
	})
	mux.HandleFunc("/ping-results", handlePingResults)
	mux.HandleFunc("/ping-results/all", getPingResults)
	mux.HandleFunc("/ping-results/latest-up-per-container", getLatestUPPerContainer)

	handler := corsMiddleware(mux)
	//Старт сервера
	fmt.Println("API is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
