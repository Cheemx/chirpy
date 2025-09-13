package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func handlerChirpsValidate(w http.ResponseWriter, r *http.Request) {
	// Request validation
	type req struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	request := req{}
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

	type response struct {
		Valid       bool   `json:"valid"`
		CleanedBody string `json:"cleaned_body"`
	}

	var errRes errorResponse
	var res response

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

	res.Valid = true
	res.CleanedBody = chirp
	data, err := json.Marshal(res)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)
}
