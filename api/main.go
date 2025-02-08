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
	Service            string    `json:"service"`
	IPAddress          string    `json:"ip_address"`
	PingDuration       float64   `json:"ping_duration"`
	Status             string    `json:"status"`
	LastSuccessfulPing time.Time `json:"last_successful_ping"`
}

func initDB() {
	var err error
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"))
	log.Println("Connecting to DB with:", connStr)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS ping_results (
		id SERIAL PRIMARY KEY,
		service TEXT,
		ip_address TEXT,
		ping_duration DOUBLE PRECISION,
		status TEXT,
		last_successful_ping TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Ошибка создания дб", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Database initialized successfully.")
}
func savePingResult(result PingResult) {
	_, err := db.Exec(
		"INSERT INTO ping_results (service, ip_address, ping_duration, status, last_successful_ping) VALUES ($1, $2, $3, $4, $5)",
		result.Service, result.IPAddress, result.PingDuration, result.Status, result.LastSuccessfulPing,
	)
	if err != nil {
		log.Printf("Error saving ping result for service %s: %v", result.Service, err)
	} else {
		log.Printf("Saved ping result for service %s", result.Service)
	}

}

func handlePingResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var result PingResult
	err := json.NewDecoder(r.Body).Decode(&result)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	savePingResult(result)
	fmt.Fprintf(w, "Ping result saved: %s - %s\n", result.Service, result.Status)
}

func getPingResults(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT service, ip_address, ping_duration, status, last_successful_ping FROM ping_results ORDER BY last_successful_ping DESC")
	if err != nil {
		http.Error(w, "Error fetching ping results", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []PingResult
	for rows.Next() {
		var result PingResult
		err := rows.Scan(&result.Service, &result.IPAddress, &result.PingDuration, &result.Status, &result.LastSuccessfulPing)
		if err != nil {
			http.Error(w, "Error scanning ping results", http.StatusInternalServerError)
			return
		}
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func main() {
	initDB()
	defer db.Close()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "заглушка")

	})
	http.HandleFunc("/ping-results", handlePingResults)
	http.HandleFunc("/ping-results/all", getPingResults)

	fmt.Println("API is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// psql -h localhost -p 5432 -U postgres -d status-check-db
