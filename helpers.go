package main

import (
		"strings"
		"slices"
		"time"
		"net/http"
		"encoding/json"
		"log"
		"github.com/skookum-ua/http-server/internal/database"
)

func censore(chirp string) string{
	veryBadWords:= []string{"kerfuffle", "sharbert", "fornax"}
	separated:= strings.Split(chirp, " ")
	for i, badword := range separated{
		if slices.Contains(veryBadWords, strings.ToLower(badword)){
			separated[i] = "****"
		}
	}
	return strings.Join(separated, " ")
}

func respondeWithError(w http.ResponseWriter, code int, msg string){
	type returnVals struct {
        CreatedAt time.Time `json:"created_at"`
		Code int `json:"code"`
		Error string `json:"error"`
	}
	  respBody := returnVals{
        CreatedAt: time.Now(),
        Code: code,
		Error: msg,
    }
	respondWithJSON(w, code, respBody)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}){
    dat, err := json.Marshal(payload)
	if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
	}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(dat)
}

func dbChirpToResponse(dbChirp database.Chirp) Chirp {
	return Chirp{
		ID:        dbChirp.ID,
		Body:      dbChirp.Body,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		UserID:    dbChirp.UserID,
	}
}