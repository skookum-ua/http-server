package main

import (
		"log"
		"encoding/json"
		"net/http"
		"fmt"
		"github.com/google/uuid"
		"github.com/skookum-ua/http-server/internal/database"
)

func handlerHealth(w http.ResponseWriter,h *http.Request){
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (c *apiConfig) handlerHits(w http.ResponseWriter,h *http.Request){
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	st := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", c.fileserverHits.Load())
	w.Write([]byte(st))
}

func (c *apiConfig) handlerReset(w http.ResponseWriter,h *http.Request){
	if c.platform != "dev" {
		log.Printf("Access denied")

			respondeWithError(w, 403,"Access denied")
			return
	}
	
	err := c.dbQueries.DeleteAllUsers(h.Context())
	if err != nil {
			log.Printf("Error deleting users: %s", err)

			respondeWithError(w, 500,"Failed to delete users")
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



func handlerValidate(w http.ResponseWriter, r *http.Request){
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
    if len(params.Body) > 140{
		respondeWithError(w, 400, "Chirp is too long")
		return
	}
	res:=response{}
	res.Body = censore(params.Body)
	respondWithJSON(w, 200, res)
}

func (c *apiConfig) handlerCreateUsers(w http.ResponseWriter, r *http.Request){
	type parameters struct {
        Email string `json:"email"`
    }

	decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		log.Printf("Error decoding parameters: %s", err)

		respondeWithError(w, 500, "Error decoding parameters")
		return
    }
	res:=User{}
	resNotNull, err := c.dbQueries.CreateUser(r.Context(), params.Email)
	if err != nil {
		log.Printf("Error creating user: %s", err)

		respondeWithError(w, 500, "Error creating user")
		return
	}
	res.ID = resNotNull.ID
	res.CreatedAt = resNotNull.CreatedAt
	res.UpdatedAt = resNotNull.UpdatedAt
	res.Email = resNotNull.Email
	respondWithJSON(w, 201, res)
}

func (c *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request){

	type parameters struct {
        Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
    }

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)
    if err != nil {
		log.Printf("Error decoding parameters: %s", err)

		respondeWithError(w, 500, "Error decoding parameters")
		return
    }
    if len(params.Body) > 140{
		respondeWithError(w, 400, "Chirp is too long")
		return
	}
	chirpParams := database.CreateChirpParams{
		Body:   censore(params.Body),
		UserID: params.UserId,
	}
	chirp, err := c.dbQueries.CreateChirp(r.Context(), chirpParams )
	if err != nil {
		log.Printf("Error creating chirp: %s", err)
		respondeWithError(w, 500, "Error creating chirp")
		return
	}

	res:=Chirp{}
	res.ID = chirp.ID
	res.CreatedAt = chirp.CreatedAt
	res.UpdatedAt = chirp.UpdatedAt
	res.Body = chirp.Body
	res.UserID = chirp.UserID
	respondWithJSON(w, 201, res)
}

func(c *apiConfig) handlerAllChirps(w http.ResponseWriter, r *http.Request){
	chirps, err := c.dbQueries.GetChirps(r.Context())
	if err != nil {
		log.Printf("Error geting chirps: %s", err)
		respondeWithError(w, 500, "Error geting chirp")
		return
	}
	newChirps := make([]Chirp, len(chirps))
	for i, chirp := range chirps{
		newChirps[i] = dbChirpToResponse(chirp)
	}
	respondWithJSON(w, 200, newChirps)
}
func(c *apiConfig) handlerGetChirpsId(w http.ResponseWriter, r *http.Request){
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
	respChirp:=dbChirpToResponse(chirp)
	respondWithJSON(w, 200, respChirp)
}


