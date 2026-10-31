package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
	"uuid"

	"github.com/statusquonjc46/bd-http-server/internal/database"
)

// /api/ endpoint handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func censorChirp(body string) string {
	bodySlice := strings.Split(body, " ")
	bad_words := map[string]bool{"kerfuffle": true, "sharbert": true, "fornax": true}
	for i, word := range bodySlice {
		if bad_words[strings.ToLower(word)] {
			bodySlice[i] = "****"
		}
	}
	return strings.Join(bodySlice, " ")
}

func respondWithErrors(w http.ResponseWriter, statusCode int, msg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}
	err := errorResponse{
		Error: msg,
	}
	respondWithPayload(w, statusCode, err)
}

func respondWithPayload(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling json %s\n", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(statusCode)
	w.Write(dat)
}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}

	type cleaned struct {
		Cleaned string `json:"cleaned_body"`
	}

	/*
		type valid struct {
			Valid bool `json:"valid"`
		}
	*/

	decoder := json.NewDecoder(r.Body)
	chirpToCheck := chirp{}

	err := decoder.Decode(&chirpToCheck)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad Request")
		return
	}

	chirpLen := len(chirpToCheck.Body)
	if chirpLen > 140 {
		log.Println(err)
		respondWithErrors(w, 400, "Chirp is too long")
		return
	}

	cleaned_body := censorChirp(chirpToCheck.Body)

	/*
		val := valid{
			Valid: true,
		}
	*/
	clean := cleaned{
		Cleaned: cleaned_body,
	}
	respondWithPayload(w, 200, clean)
}

func (cfg *apiConfig) addUserHandler(w http.ResponseWriter, r *http.Request) {
	type reqEmail struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	emailBody := reqEmail{}

	err := decoder.Decode(&emailBody)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad Request")
		return
	}

	user, err := cfg.database.CreateUser(r.Context(), emailBody.Email)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to add user")
		return
	}
	jsonUser := User{
		ID:        uuid.UUID(user.ID),
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
		Email:     user.Email,
	}
	respondWithPayload(w, 201, jsonUser)
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

// API Config Related Handlers at /admin/ endpoints
type apiConfig struct {
	fileserverHits atomic.Int32
	database       *database.Queries
	platform       string
}

func (cfg *apiConfig) middlewareMetricsIncrement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricHandler(w http.ResponseWriter, r *http.Request) {
	hitsCount := cfg.fileserverHits.Load()
	hitsString := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", hitsCount)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(hitsString))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithErrors(w, 403, "Forbidden")
	}
	err := cfg.database.DeleteAllUsers(r.Context())
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to reset table")
		return
	}
	cfg.fileserverHits.Store(0)
}
