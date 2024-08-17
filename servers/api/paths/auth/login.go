package auth

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/api"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"time"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var body LoginBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	password, err := api.Decrypt(body.Password, api.PrePassphrase)
	if err != nil {
		http.Error(w, "Failed to decrypt password", http.StatusInternalServerError)
		return
	}

	if password == api.ApiPassword {
		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)
		claims["username"] = body.Username
		claims["exp"] = time.Now().Add(time.Hour * 1).Unix()
		tokenString, err := token.SignedString(api.PrePassphrase)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(LoginResponse{Token: tokenString})
	} else {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	}
}
