package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/skookum-ua/http-server/internal/auth"
	"github.com/skookum-ua/http-server/internal/database"
)

func handlerHealth(w http.ResponseWriter, h *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (c *apiConfig) handlerHits(w http.ResponseWriter, h *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	st := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", c.fileserverHits.Load())
	w.Write([]byte(st))
}

func (c *apiConfig) handlerReset(w http.ResponseWriter, h *http.Request) {
	if c.platform != "dev" {
		log.Printf("Access denied")

		respondeWithError(w, 403, "Access denied")
		return
	}

	err := c.dbQueries.DeleteAllUsers(h.Context())
	if err != nil {
		log.Printf("Error deleting users: %s", err)

		respondeWithError(w, 500, "Failed to delete users")
		return
	}
	c.fileserverHits.Swap(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("Hits reseted"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func handlerValidate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type response struct {
		Body string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)

		respondeWithError(w, 500, "Error decoding parameters")
		return
	}
	if len(params.Body) > 140 {
		respondeWithError(w, 400, "Chirp is too long")
		return
	}
	res := response{}
	res.Body = censore(params.Body)
	respondWithJSON(w, 200, res)
}

func (c *apiConfig) handlerCreateUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)

		respondeWithError(w, 500, "Error decoding parameters")
		return
	}

	hashedParams := database.CreateUserParams{}
	hashedParams.Email = params.Email
	hashedParams.HashedPasswords, err = auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error hadhing password: %s", err)
		respondeWithError(w, 500, "Error hadhing password")
		return
	}
	resNotNull, err := c.dbQueries.CreateUser(r.Context(), hashedParams)
	if err != nil {
		log.Printf("Error creating user: %s", err)

		respondeWithError(w, 500, "Error creating user")
		return
	}
	res := User{}
	res.ID = resNotNull.ID
	res.CreatedAt = resNotNull.CreatedAt
	res.UpdatedAt = resNotNull.UpdatedAt
	res.Email = resNotNull.Email
	respondWithJSON(w, 201, res)
}

func (c *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Body string `json:"body"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error geting token: %s", err)

		respondeWithError(w, 401, "Error geting token")
		return
	}

	id, err := auth.ValidateJWT(token, c.secret)
	if err != nil {
		log.Printf("Error validating token: %s", err)

		respondeWithError(w, 401, "Error validating token")
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)

		respondeWithError(w, 500, "Error decoding parameters")
		return
	}
	if len(params.Body) > 140 {
		respondeWithError(w, 400, "Chirp is too long")
		return
	}
	chirpParams := database.CreateChirpParams{
		Body:   censore(params.Body),
		UserID: id,
	}
	chirp, err := c.dbQueries.CreateChirp(r.Context(), chirpParams)
	if err != nil {
		log.Printf("Error creating chirp: %s", err)
		respondeWithError(w, 500, "Error creating chirp")
		return
	}

	res := Chirp{}
	res.ID = chirp.ID
	res.CreatedAt = chirp.CreatedAt
	res.UpdatedAt = chirp.UpdatedAt
	res.Body = chirp.Body
	res.UserID = chirp.UserID
	respondWithJSON(w, 201, res)
}

func (c *apiConfig) handlerAllChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := c.dbQueries.GetChirps(r.Context())
	if err != nil {
		log.Printf("Error geting chirps: %s", err)
		respondeWithError(w, 500, "Error geting chirp")
		return
	}
	newChirps := make([]Chirp, len(chirps))
	for i, chirp := range chirps {
		newChirps[i] = dbChirpToResponse(chirp)
	}
	respondWithJSON(w, 200, newChirps)
}
func (c *apiConfig) handlerGetChirpsId(w http.ResponseWriter, r *http.Request) {
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondeWithError(w, 400, "Invalid chirp ID")
		return
	}
	chirp, err := c.dbQueries.GetsChirpsId(r.Context(), chirpID)
	if err != nil {
		respondeWithError(w, 404, "No chirp found")
		return
	}
	respChirp := dbChirpToResponse(chirp)
	respondWithJSON(w, 200, respChirp)
}

func (c *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	expiresIn := 1 * time.Hour

	refresh_token := auth.MakeRefreshToken()

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)

		respondeWithError(w, 401, "Error decoding parameters")
		return
	}

	user, err := c.dbQueries.GetUserByID(r.Context(), params.Email)
	if err != nil {
		log.Printf("No registered user with such email: %s", err)

		respondeWithError(w, 401, "No registered user with such email")
		return
	}

	passCheck, err := auth.CheckPasswordHash(params.Password, user.HashedPasswords)
	if err != nil || passCheck == false {
		log.Printf("Error checking password: %s", err)
		respondeWithError(w, 401, "Error checking password")
		return
	}

	secret := c.secret

	tokenString, err := auth.MakeJWT(user.ID, secret, expiresIn)
	if err != nil {
		log.Printf("Error creating token: %s", err)

		respondeWithError(w, 500, "Error creating token")
		return
	}
	rfTokkenParams := database.CreateRefreshTokenParams{}
	rfTokkenParams.Token = refresh_token
	rfTokkenParams.ExpiresAt = time.Now().Add(24 * time.Hour)
	rfTokkenParams.UserID = user.ID

	c.dbQueries.CreateRefreshToken(r.Context(), rfTokkenParams)

	res := User{}
	res.ID = user.ID
	res.CreatedAt = user.CreatedAt
	res.UpdatedAt = user.UpdatedAt
	res.Email = user.Email
	res.Token = tokenString
	res.RefToken = refresh_token
	respondWithJSON(w, 200, res)
}

func (c *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	ref_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error checking token: %s", err)
		respondeWithError(w, 401, "Error checking token")
		return
	}
	db_ref_tokrn, err := c.dbQueries.GetRefreshToken(r.Context(), ref_token)
	if err != nil {
		log.Printf("Error checking token: %s", err)
		respondeWithError(w, 401, "Error checking token")
		return
	}
	if time.Now().After(db_ref_tokrn.ExpiresAt) || db_ref_tokrn.RevokedAt.Valid {
		log.Printf("Token expired or revoked")
		respondeWithError(w, 401, "Token expired or revoked")
		return
	}

	tokenString, err := auth.MakeJWT(db_ref_tokrn.UserID, c.secret, time.Hour)
	if err != nil {
		log.Printf("Error creating token: %s", err)

		respondeWithError(w, 401, "Error creating token")
		return
	}

	type response struct {
		Token string `json:"token"`
	}

	res := response{}
	res.Token = tokenString
	respondWithJSON(w, 200, res)
}

func (c *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	ref_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error checking token: %s", err)
		respondeWithError(w, 500, "Error checking token")
		return
	}

	revTokPar := database.RevokeTokenParams{}
	revTokPar.Token = ref_token
	revTokPar.RevokedAt = sql.NullTime{Time: time.Now(), Valid: true}
	revTokPar.UpdatedAt = time.Now()
	err = c.dbQueries.RevokeToken(r.Context(),revTokPar)
	if err != nil {
		log.Printf("Error revoking token: %s", err)
		respondeWithError(w, 500, "Error revoking token")
		return
	}
    w.WriteHeader(204)
}
