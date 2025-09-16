package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/Cheemx/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// atomic.Int32 is a really cool standard-library type
// that allows us to safely increment and read an integer value
// across multiple goroutines (HTTP requests).
type apiConfig struct {
	fileServerHits atomic.Int32
	db             *database.Queries
	user           *database.User
	jwtSecret      string
	polkaKey       string
}

const (
	port         = "8080"
	filePathRoot = "."
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("Can't get DB_URL from .env")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	mux := &http.ServeMux{}
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	cfg := &apiConfig{
		fileServerHits: atomic.Int32{},
		db:             dbQueries,
		jwtSecret:      os.Getenv("SECRET"),
		polkaKey:       os.Getenv("POLKA_KEY"),
	}

	// /app route handler to increment hits
	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir(filePathRoot)))
	wrapped := cfg.middlewareMetricsInc(appHandler)
	mux.Handle("/app/", wrapped)
	mux.Handle("/app", wrapped)

	// API health checker
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("OK"))
	})

	// metrics tracking API
	mux.HandleFunc("GET /admin/metrics", cfg.handleMetrics)

	// reset fileServerHits in cfg
	mux.HandleFunc("POST /admin/reset", cfg.handleReset)

	// Create Chirp endpoint
	mux.HandleFunc("POST /api/chirps", cfg.handleCreateChirp)

	// Create User endpoint
	mux.HandleFunc("POST /api/users", cfg.handleCreateUser)

	// Update User endpoint
	mux.HandleFunc("PUT /api/users", cfg.handleUpdateUser)

	// Login User endpoint
	mux.HandleFunc("POST /api/login", cfg.handleLoginUser)

	// Check Token Expiry endpoint
	mux.HandleFunc("POST /api/refresh", cfg.handleRefreshToken)

	// Revoke the Refresh Token
	mux.HandleFunc("POST /api/revoke", cfg.handleRevokeRefreshToken)

	// Update User to Red Endpoint
	mux.HandleFunc("POST /api/polka/webhooks", cfg.handleUpdateUserToRed)

	// Get AllChirps endpoint
	mux.HandleFunc("GET /api/chirps", cfg.handleGetChirps)

	// Get Chirp by ID
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.handleGetChirpByID)

	// Delete Chirp by ID
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.handleDeleteChirp)

	// Starting the Server
	log.Printf("Serving files from %s on port: %s\n", filePathRoot, port)
	log.Fatal(server.ListenAndServe())
}
