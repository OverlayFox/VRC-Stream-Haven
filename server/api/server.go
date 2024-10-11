package api

import (
	"github.com/OverlayFox/VRC-Stream-Haven/ingest/api/paths/auth"
	"github.com/OverlayFox/VRC-Stream-Haven/ingest/api/paths/escort"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

var PrePassphrase []byte
var Password string

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
			return PrePassphrase, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// localMiddleware checks if the request is coming from localhost
func localMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if !strings.HasPrefix(ip, "127.0.0.1") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// InitApi initializes the mux router and sets up the routes
func InitApi() *mux.Router {
	r := mux.NewRouter()
	// frontfacing API
	r.HandleFunc("/auth/login", auth.LoginHandler).Methods("POST")
	r.Handle("/escort/register", jwtMiddleware(http.HandlerFunc(escort.RegisterHandler))).Methods("POST")

	// backfacing API
	//r.Handle("/auth/ingest", localMiddleware(http.HandlerFunc(authIngest))).Methods("POST")
	//r.Handle("/ingest/receive", localMiddleware(http.HandlerFunc(ingestReceive))).Methods("POST")

	return r
}
