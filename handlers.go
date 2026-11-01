package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	//"uuid"

	"github.com/google/uuid"
	"github.com/statusquonjc46/bd-http-server/internal/auth"
	"github.com/statusquonjc46/bd-http-server/internal/database"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

// HELPER FUNCTIONS
// helper function for censoring chirps
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

// helper function for creating error http responses
func respondWithErrors(w http.ResponseWriter, statusCode int, msg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}
	err := errorResponse{
		Error: msg,
	}
	respondWithPayload(w, statusCode, err)
}

// helper function for creating json reponse payloads
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

// API FUNCTIONS
// api function to handle validation and insertion of a chirp to chirps table
func (cfg *apiConfig) chirpHandler(w http.ResponseWriter, r *http.Request) {
	type reqChirp struct {
		Body    string        `json:"body"`
		User_Id uuid.NullUUID `json:"user_id"`
	}

	type cleaned struct {
		Cleaned string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	chirpToCheck := reqChirp{}

	err := decoder.Decode(&chirpToCheck)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad Request")
		return
	}

	bearer, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad request - Token")
		return
	}

	jwtUserId, err := auth.ValidateJWT(bearer, cfg.bearer)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 401, "Unauthorized")
		return
	}

	validUUID := uuid.NullUUID{
		UUID:  jwtUserId,
		Valid: true,
	}

	chirpLen := len(chirpToCheck.Body)
	if chirpLen > 140 {
		log.Println(err)
		respondWithErrors(w, 400, "Chirp is too long")
		return
	}

	cleaned_body := censorChirp(chirpToCheck.Body)

	argChirp := database.CreateChirpParams{
		Body:   cleaned_body,
		UserID: validUUID,
	}

	createdChirp, err := cfg.database.CreateChirp(r.Context(), argChirp)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to create chirp in database")
		return
	}
	payloadChirp := Chirp{
		ID:        createdChirp.ID,
		CreatedAt: createdChirp.CreatedAt.Time,
		UpdatedAt: createdChirp.UpdatedAt.Time,
		Body:      createdChirp.Body,
		UserID:    createdChirp.UserID,
	}
	respondWithPayload(w, 201, payloadChirp)
}

// api function to get all chirps
func (cfg *apiConfig) getAllChirpsHandler(w http.ResponseWriter, r *http.Request) {
	chirpSlice := []Chirp{}
	authId := r.URL.Query().Get("author_id")
	sorting := r.URL.Query().Get("sort")
	var allChirps []database.Chirp

	if authId == "" {
		returnedChirps, err := cfg.database.GetAllChirps(r.Context())
		if err != nil {
			log.Println(err)
			respondWithErrors(w, 500, "Failed to query database for all chirps")
			return
		}
		allChirps = returnedChirps
	} else {
		authUUID, err := uuid.Parse(authId)
		if err != nil {
			log.Println(err)
			respondWithErrors(w, 400, "Bad request - author id")
		}
		returnedChirps, err := cfg.database.GetAllChirpsByAuthor(r.Context(), uuid.NullUUID{UUID: authUUID, Valid: true})
		if err != nil {
			log.Println(err)
			respondWithErrors(w, 404, "Author ID not present")
			return
		}
		allChirps = returnedChirps
	}

	for _, chirp := range allChirps {
		appendChirp := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt.Time,
			UpdatedAt: chirp.UpdatedAt.Time,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}

		chirpSlice = append(chirpSlice, appendChirp)
	}
	if sorting == "desc" {
		slices.SortFunc(chirpSlice, func(a, b Chirp) int {
			return b.CreatedAt.Compare(a.CreatedAt)
		})
	}
	respondWithPayload(w, 200, chirpSlice)
}

// api function to get a single postgres delete row from tablechirp by chirp id
func (cfg *apiConfig) getAChirp(w http.ResponseWriter, r *http.Request) {
	type chirpId struct {
		ID uuid.UUID
	}
	pathVal := r.PathValue("chirpId")
	parsedVal, err := uuid.Parse(pathVal)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad request")
		return
	}

	returnedChirp, err := cfg.database.GetAChirp(r.Context(), parsedVal)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 404, "Failed to query database for chirp")
		return
	}

	payloadChirp := Chirp{
		ID:        returnedChirp.ID,
		CreatedAt: returnedChirp.CreatedAt.Time,
		UpdatedAt: returnedChirp.UpdatedAt.Time,
		Body:      returnedChirp.Body,
		UserID:    returnedChirp.UserID,
	}

	respondWithPayload(w, 200, payloadChirp)

}

// api function to delete a chirp is authenticated as the authoer
func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		ChirpID uuid.UUID `json:"id"`
		UserID  uuid.NullUUID
	}

	pathVal := r.PathValue("chirpId")
	parsedVal, err := uuid.Parse(pathVal)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad request")
		return
	}

	bearer, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 401, "Unauthorized")
		return
	}

	userId, err := auth.ValidateJWT(bearer, cfg.bearer)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 401, "Unauthorized")
		return
	}

	delParams := database.DeleteAChirpParams{
		ID: parsedVal,
		UserID: uuid.NullUUID{
			UUID:  userId,
			Valid: true,
		},
	}

	returnedChirp, err := cfg.database.GetAChirp(r.Context(), delParams.ID)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 404, "Chirp not found")
		return
	}

	if returnedChirp.ID != parsedVal || returnedChirp.UserID.UUID != userId {
		log.Println(err)
		respondWithErrors(w, 403, "Forbidden")
		return
	}

	err = cfg.database.DeleteAChirp(r.Context(), delParams)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to delete chirp")
		return
	}

	w.WriteHeader(204)
}

// api function to handle adding a user to users table
func (cfg *apiConfig) addUserHandler(w http.ResponseWriter, r *http.Request) {
	type reqEmail struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	reqBody := reqEmail{}

	err := decoder.Decode(&reqBody)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad Request")
		return
	}

	reqBody.Password, err = auth.HashPassword(reqBody.Password)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to hash password")
		return
	}

	userArgs := database.CreateUserParams{
		Email:          reqBody.Email,
		HashedPassword: reqBody.Password,
	}

	user, err := cfg.database.CreateUser(r.Context(), userArgs)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to add user")
		return
	}
	jsonUser := User{
		ID:          uuid.UUID(user.ID),
		CreatedAt:   user.CreatedAt.Time,
		UpdatedAt:   user.UpdatedAt.Time,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed.Bool,
	}
	respondWithPayload(w, 201, jsonUser)
}

// api function to update a user email and password if authenticated
func (cfg *apiConfig) updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	type newCreds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	reqBody := newCreds{}

	err := decoder.Decode(&reqBody)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad Request")
		return
	}

	bearer, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 401, "Unauthorized")
		return
	}

	userId, err := auth.ValidateJWT(bearer, cfg.bearer)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 401, "Unauthorized")
		return
	}

	userData, err := cfg.database.GetUserByID(r.Context(), userId)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "UserId does not exist")
		return
	}

	hashed, err := auth.HashPassword(reqBody.Password)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to hash password")
	}

	userParams := database.UpdateUserCredsParams{
		Email:          reqBody.Email,
		HashedPassword: hashed,
		ID:             userData.ID,
	}

	dat, err := cfg.database.UpdateUserCreds(r.Context(), userParams)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to update email or password")
	}
	payloadUser := User{
		ID:          dat.ID,
		CreatedAt:   dat.CreatedAt.Time,
		UpdatedAt:   dat.UpdatedAt.Time,
		Email:       dat.Email,
		IsChirpyRed: dat.IsChirpyRed.Bool,
	}
	respondWithPayload(w, 200, payloadUser)
}

// api function to handle a login attempt via email and password to the users table
func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	type creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	credBody := creds{}

	err := decoder.Decode(&credBody)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad request")
		return
	}

	user, err := cfg.database.ValidateEmail(r.Context(), credBody.Email)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 401, "incorrect email or password")
	}

	match, err := auth.CheckPasswordHash(credBody.Password, user.HashedPassword)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "failed to check validate password")
		return
	}

	if match != true {
		respondWithErrors(w, 401, "incorrect email or password")
		return
	}

	payloadUser := UserWithToken{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt.Time,
		UpdatedAt:   user.UpdatedAt.Time,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed.Bool,
	}

	userToken, err := auth.MakeJWT(payloadUser.ID, cfg.bearer, accessExpiresAt)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to create token")
		return
	}

	refreshParams := database.InsertTokenParams{
		Token:  auth.MakeRefreshToken(),
		UserID: uuid.NullUUID{UUID: payloadUser.ID, Valid: true},
	}
	refreshToken, err := cfg.database.InsertToken(r.Context(), refreshParams)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to create refresh token")
		return
	}

	payloadUser.Token = userToken
	payloadUser.Refresh = refreshToken.Token

	respondWithPayload(w, 200, payloadUser)
}

// api function to refresh a access token if refresh token is valid
func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, r *http.Request) {
	type accessToken struct {
		Token string `json:"token"`
	}
	bearer, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad request - bearer")
		return
	}
	tokenData, err := cfg.database.ValidateRefreshToken(r.Context(), bearer)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 401, "Unauthorized")
		return
	}

	if tokenData.ExpiresAt.Time.Before(time.Now()) || tokenData.RevokedAt.Valid {
		respondWithErrors(w, 401, "Unauthorized")
		return
	}
	newToken, err := auth.MakeJWT(tokenData.UserID.UUID, cfg.bearer, accessExpiresAt)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to create new access token from bearer")
	}

	payloadToken := accessToken{
		Token: newToken,
	}
	respondWithPayload(w, 200, payloadToken)

}

// api function to revoke the refresh token
func (cfg *apiConfig) revokeRefreshHandler(w http.ResponseWriter, r *http.Request) {
	bearer, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad request - bearer")
		return
	}
	err = cfg.database.RevokeRefreshToken(r.Context(), bearer)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 401, "Unauthorized")
	}
	w.WriteHeader(204)
}

// api middleware to count page visits
func (cfg *apiConfig) middlewareMetricsIncrement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// api function to return the visit count
func (cfg *apiConfig) metricHandler(w http.ResponseWriter, r *http.Request) {
	hitsCount := cfg.fileserverHits.Load()
	hitsString := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", hitsCount)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(hitsString))
}

// api function to reset the count of visits and reset the users table
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

// api function to update a user to chirpy red
func (cfg *apiConfig) updateChirpyRed(w http.ResponseWriter, r *http.Request) {
	type eventUpdate struct {
		Event string `json:"event"`
		Data  struct {
			UserId uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	eventBody := eventUpdate{}

	err := decoder.Decode(&eventBody)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 400, "Bad request")
		return
	}

	apiKey, err := auth.GetApiKey(r.Header)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 401, "Unauthorized")
		return
	}

	if apiKey != cfg.polka {
		respondWithErrors(w, 401, "Unauthorized")
		return
	}

	if eventBody.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	redParams := database.UpdateChirpyRedParams{
		IsChirpyRed: sql.NullBool{Bool: true, Valid: true},
		ID:          eventBody.Data.UserId,
	}
	userData, err := cfg.database.UpdateChirpyRed(r.Context(), redParams)
	if err != nil {
		log.Println(err)
		respondWithErrors(w, 404, "User not found")
		return
	}

	if userData.IsChirpyRed.Bool == false {
		log.Println(err)
		respondWithErrors(w, 500, "Failed to update chirpy status")
		return
	}

	w.WriteHeader(204)
}
