package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/Cheemx/chirpy/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Request Validation
	type req struct {
		Email string `json:"email"`
	}

	var request req
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("Error decoding request: %v", err)
		w.WriteHeader(500)
		return
	}

	// Response Validation
	user, err := cfg.db.CreateUser(r.Context(), request.Email)
	if err != nil {
		log.Printf("Error creating User: %v", err)
		w.WriteHeader(500)
		return
	}

	cfg.user = &user

	// Creating response and responding
	data, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		w.WriteHeader(500)
		return
	}

	// Responding!
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(data)
}

func (cfg *apiConfig) handleCreateChirp(w http.ResponseWriter, r *http.Request) {
	// Request validation
	decoder := json.NewDecoder(r.Body)
	request := database.CreateChirpParams{}
	err := decoder.Decode(&request)
	if err != nil {
		log.Printf("Error decoding request parameters: %s", err)
		w.WriteHeader(500)
		return
	}

	// Response handling
	type errorResponse struct {
		Error string `json:"error"`
	}

	var errRes errorResponse

	// Sending error for Chirp length out of range
	if len(request.Body) > 140 {
		errRes.Error = "Chirp is too long"
		data, err := json.Marshal(errRes)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write(data)
		return
	}

	// replacing "profane" words with "****" in request.Body
	words := strings.Fields(request.Body)
	clean := []string{}
	for _, word := range words {
		lowerWord := strings.ToLower(word)
		if lowerWord == "kerfuffle" || lowerWord == "sharbert" || lowerWord == "fornax" {
			clean = append(clean, "****")
		} else {
			clean = append(clean, word)
		}
	}
	chirp := strings.Join(clean, " ")

	// Creating the Chirp in database
	request.Body = chirp
	request.UserID = cfg.user.ID
	Chirp, err := cfg.db.CreateChirp(r.Context(), request)
	if err != nil {
		log.Printf("Error Creating Chirp: %v", err)
		w.WriteHeader(500)
		return
	}

	// Creating response Body
	data, err := json.Marshal(Chirp)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		w.WriteHeader(500)
		return
	}

	// Responding!
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(data)
}

func (cfg *apiConfig) handleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetAllChirps(r.Context())
	if err != nil {
		log.Printf("Error getting Chirps: %v", err)
		w.WriteHeader(500)
		return
	}

	// Creating response Body
	data, err := json.Marshal(chirps)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		w.WriteHeader(500)
		return
	}

	// Responding!
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)
}

func (cfg *apiConfig) handleGetChirpByID(w http.ResponseWriter, r *http.Request) {
	ChirpID := r.PathValue("chirpID")

	chirp, err := cfg.db.GetChirpByID(r.Context(), uuid.MustParse(ChirpID))
	if err != nil {
		if err == sql.ErrNoRows {
			log.Print("Chirp not found")
			w.WriteHeader(404)
			return
		}
		log.Printf("Error getting the Chirp: %v", err)
		w.WriteHeader(500)
		return
	}

	// Creating response Body
	data, err := json.Marshal(chirp)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		w.WriteHeader(500)
		return
	}

	// Responding!
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)
}
