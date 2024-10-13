package server

import (
	"github.com/OverlayFox/VRC-Stream-Haven/api/server/escort/paths/info"
	"github.com/OverlayFox/VRC-Stream-Haven/api/server/flagship/paths/register"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"net/http"
	"os"
)

var Passphrase = []byte(os.Getenv("PASSPHRASE"))

// jwtMiddleware checks for a valid JWT token in the Authorization header
func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Token is missing", http.StatusUnauthorized)
			return
		}

		tokenString = tokenString[len("Bearer "):]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return Passphrase, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// InitFlagshipApi initializes the mux router and sets up the routes for a Flagship Endpoint.
func InitFlagshipApi() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/flagship/register", jwtMiddleware(http.HandlerFunc(register.PostRegisterEscortToHaven))).Methods("POST")

	return r
}

// InitEscortApi initializes the mux router and sets up the routes for a Escort Endpoint.
func InitEscortApi() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/escort/info", jwtMiddleware(http.HandlerFunc(info.GetInfo))).Methods("GET")

	return r
}
