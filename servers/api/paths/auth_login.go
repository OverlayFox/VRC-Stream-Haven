package paths

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/api"
	"github.com/dgrijalva/jwt-go"
	"io"
	"net/http"
	"time"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bodyJson, err := api.Decrypt(string(bodyBytes), api.PrePassphrase)
	if err != nil {
		http.Error(w, "Failed to decrypt body", http.StatusInternalServerError)
		return
	}

	var body LoginBody
	if err := json.Unmarshal([]byte(bodyJson), &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if body.Password == api.Password {
		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)
		claims["username"] = body.Username
		claims["exp"] = time.Now().Add(time.Hour * 1).Unix()
		tokenString, err := token.SignedString(api.PrePassphrase)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		jsonData, err := json.Marshal(LoginResponse{Token: tokenString})
		if err != nil {
			http.Error(w, "Failed to generate response", http.StatusInternalServerError)
			return
		}

		encryptedData, err := api.Encrypt(string(jsonData), api.PrePassphrase)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(encryptedData))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	}
}
