package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	_ "github.com/lib/pq"
)

type Response struct {
	Status   string `json:"status,omitempty"`
	Service  string `json:"service"`
	Message  string `json:"message,omitempty"`
	Database string `json:"database,omitempty"`
}

type Task struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func sendJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func allowedOrigin(origin string) bool {
	if origin == "http://localhost:14200" || origin == "http://127.0.0.1:14200" || origin == "http://localhost:4200" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Scheme == "https" &&
		strings.HasSuffix(parsed.Hostname(), "-frontend.valadeploy.internal") &&
		(parsed.Port() == "" || parsed.Port() == "443")
}

func routes(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	health := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		database := "not configured"
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := db.PingContext(ctx); err != nil {
				sendJSON(w, http.StatusServiceUnavailable, Response{Status: "degraded", Service: "go-backend", Database: "offline"})
				return
			}
			database = "online"
		}
		sendJSON(w, http.StatusOK, Response{Status: "online", Service: "go-backend", Database: database})
	}

	mux.HandleFunc("/health", health)
	mux.HandleFunc("/api/health", health)
	mux.HandleFunc("/api/v1/health", health)
	mux.HandleFunc("/api/v1/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		sendJSON(w, http.StatusOK, Response{Service: "go-backend", Message: "Bonjour depuis Go et PostgreSQL !"})
	})

	mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			http.Error(w, "Database not configured", http.StatusServiceUnavailable)
			return
		}
		switch r.Method {
		case http.MethodGet:
			rows, err := db.QueryContext(r.Context(), "SELECT id, title, done FROM tasks ORDER BY id DESC LIMIT 100")
			if err != nil {
				log.Printf("read tasks: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			tasks := make([]Task, 0)
			for rows.Next() {
				var task Task
				if err := rows.Scan(&task.ID, &task.Title, &task.Done); err != nil {
					log.Printf("scan task: %v", err)
					http.Error(w, "Database error", http.StatusInternalServerError)
					return
				}
				tasks = append(tasks, task)
			}
			if err := rows.Err(); err != nil {
				log.Printf("list tasks: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			sendJSON(w, http.StatusOK, tasks)

		case http.MethodPost:
			r.Body = http.MaxBytesReader(w, r.Body, 4096)
			var payload struct {
				Title string `json:"title"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&payload); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			payload.Title = strings.TrimSpace(payload.Title)
			if payload.Title == "" || utf8.RuneCountInString(payload.Title) > 120 {
				http.Error(w, "Title must have 1-120 characters", http.StatusBadRequest)
				return
			}
			task := Task{Title: payload.Title}
			err := db.QueryRowContext(r.Context(), "INSERT INTO tasks (title) VALUES ($1) RETURNING id, done", task.Title).Scan(&task.ID, &task.Done)
			if err != nil {
				log.Printf("create task: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			sendJSON(w, http.StatusCreated, task)

		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			if !allowedOrigin(origin) {
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func openDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)

	for attempt := 1; attempt <= 30; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			break
		}
		log.Printf("PostgreSQL not ready (%d/30): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("PostgreSQL connection: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
        id BIGSERIAL PRIMARY KEY,
        title VARCHAR(120) NOT NULL,
        done BOOLEAN NOT NULL DEFAULT FALSE
    )`)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("schema initialization: %w", err)
	}
	return db, nil
}

func main() {
	db, err := openDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           routes(db),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Printf("Go API with PostgreSQL ready on :%s", port)
	log.Fatal(server.ListenAndServe())
}
