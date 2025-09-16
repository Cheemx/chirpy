package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Cheemx/chirpy/internal/auth"
	"github.com/Cheemx/chirpy/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Request Validation
	req := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding request: %v", err)
		w.WriteHeader(500)
		return
	}

	// Hashing the normal text password from r.Body
	hashedPass, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing the password: %v", err)
		w.WriteHeader(500)
		return
	}

	// Response Validation
	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          req.Email,
		HashedPassword: hashedPass,
	})
	if err != nil {
		log.Printf("Error creating User: %v", err)
		w.WriteHeader(500)
		return
	}

	cfg.user = &user

	// Creating response and responding
	res := struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}
	data, err := json.Marshal(res)
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

func (cfg *apiConfig) handleLoginUser(w http.ResponseWriter, r *http.Request) {
	// request parsing
	req := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		ExpireIn int    `json:"expires_in_seconds,omitempty"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding request: %v", err)
		w.WriteHeader(500)
		return
	}

	// Get User by Email
	user, err := cfg.db.GetUserAndHashPassByEmail(r.Context(), req.Email)
	if err != nil {
		log.Printf("Incorrect email or password: %v", err)
		w.WriteHeader(401)
		return
	}

	// validating password
	err = auth.CheckPasswordHash(req.Password, user.HashedPassword)
	if err != nil {
		log.Print(err)
		w.WriteHeader(401)
		return
	}

	// Create the Access Token
	if req.ExpireIn == 0 || req.ExpireIn > 3600 {
		req.ExpireIn = 3600
	}

	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Duration(req.ExpireIn)*time.Second)
	if err != nil {
		log.Printf("Error making JWT %v", err)
		w.WriteHeader(500)
		return
	}

	// Create the Refresh Token
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("Error making Refresh Token %v", err)
		w.WriteHeader(500)
		return
	}

	refTok, err := cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: user.ID,
	})
	if err != nil {
		log.Printf("Error storing refresh token %v", err)
		w.WriteHeader(500)
		return
	}

	// Creating response and responding
	res := struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refTok.Token,
	}
	data, err := json.Marshal(res)
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

func (cfg *apiConfig) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	// get token for header
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Header token error %v", err)
		w.WriteHeader(500)
		return
	}

	// Look up token in db
	refToken, err := cfg.db.GetTokenByTokenValue(r.Context(), token)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Print("Error No Rows with specified token")
			w.WriteHeader(401)
			return
		}
		log.Printf("Error getting refToken %v", err)
		w.WriteHeader(500)
		return
	}
	// if expired return 401
	if time.Now().Compare(refToken.ExpiresAt) > 0 {
		log.Print("token expired")
		w.WriteHeader(401)
		return
	}

	// check if revoked
	if refToken.RevokedAt.Valid {
		log.Print("token revoked")
		w.WriteHeader(401)
		return
	}

	// Get the user of that token
	user, err := cfg.db.GetUserFromRefreshToken(r.Context(), refToken.Token)
	if err != nil {
		log.Printf("Error getting user from token value %v", err)
		w.WriteHeader(500)
		return
	}

	// Create new access token for the user
	accessToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, 3600*time.Second)
	if err != nil {
		log.Printf("Error making JWT %v", err)
		w.WriteHeader(500)
		return
	}

	// Creating response
	res := struct {
		Token string `json:"token"`
	}{
		Token: accessToken,
	}
	data, err := json.Marshal(res)
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

func (cfg *apiConfig) handleRevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error making Refresh Token %v", err)
		w.WriteHeader(401)
		return
	}

	// Update the token in database
	rowsAffected, err := cfg.db.SetTokenTimestamps(r.Context(), database.SetTokenTimestampsParams{
		RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt: time.Now(),
		Token:     token,
	})
	if err != nil {
		log.Printf("Error setting Refresh Token timestamps %v", err)
		w.WriteHeader(500)
		return
	}

	if rowsAffected < 1 {
		log.Printf("Error setting Refresh Token timestamps %v", err)
		w.WriteHeader(401)
		return
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) handleCreateChirp(w http.ResponseWriter, r *http.Request) {
	// Authorization checkpoint
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Auth Header error: %v", err)
		w.WriteHeader(401)
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Printf("JWT validation error: %v", err)
		w.WriteHeader(401)
		return
	}
	// Request validation
	decoder := json.NewDecoder(r.Body)
	request := database.CreateChirpParams{}
	err = decoder.Decode(&request)
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
	request.UserID = userId
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

func (cfg *apiConfig) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	// get the authorization header
	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting auth Header %v", err)
		w.WriteHeader(401)
		return
	}

	// get the user from the token
	userID, err := auth.ValidateJWT(accessToken, cfg.jwtSecret)
	if err != nil {
		log.Printf("Error getting user from token value %v", err)
		w.WriteHeader(401)
		return
	}

	// /request body to parse the http request
	req := struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}{}

	// Decode the request
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Print("Error unmarshalling the request")
		w.WriteHeader(500)
		return
	}

	// validate email and password for emptiness
	if req.Email == "" || req.Password == "" {
		log.Print("Empty request parameters")
		w.WriteHeader(401)
		return
	}

	// hash the text password
	hashedPass, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Print("Error unmarshalling the request")
		w.WriteHeader(500)
		return
	}

	// update the user with new information
	updateduser, err := cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          req.Email,
		HashedPassword: hashedPass,
		ID:             userID,
	})
	if err != nil {
		log.Printf("Error updating the user %v", err)
		w.WriteHeader(500)
		return
	}

	// Creating response and responding
	res := struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}{
		ID:        updateduser.ID,
		CreatedAt: updateduser.CreatedAt,
		UpdatedAt: updateduser.UpdatedAt,
		Email:     updateduser.Email,
	}
	data, err := json.Marshal(res)
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

func (cfg *apiConfig) handleDeleteChirp(w http.ResponseWriter, r *http.Request) {
	// get the authorization header
	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting auth Header %v", err)
		w.WriteHeader(401)
		return
	}

	// get the user from the token
	userID, err := auth.ValidateJWT(accessToken, cfg.jwtSecret)
	if err != nil {
		log.Printf("Error getting user from token value %v", err)
		w.WriteHeader(401)
		return
	}

	// Get the chitpID from request parameters
	ChirpID := r.PathValue("chirpID")

	// parse the chirpID to uuid format
	id, err := uuid.Parse(ChirpID)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	// Get Chirp from Database by ID
	chirp, err := cfg.db.GetChirpByID(r.Context(), id)
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

	// Check if Authenticated user and Chirp's author are same or not
	if chirp.UserID != userID {
		log.Print("Chirp Author and current user are not same")
		w.WriteHeader(403)
		return
	}

	// Delete the chirp since user and chirpID are confirmed
	err = cfg.db.DeleteChirpByID(r.Context(), uuid.MustParse(ChirpID))
	if err != nil {
		log.Printf("Error deleting the Chirp: %v", err)
		w.WriteHeader(500)
		return
	}

	// chirp deleted successfully
	w.WriteHeader(204)
}
