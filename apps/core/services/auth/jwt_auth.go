package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTAuthenticator struct {
	secretKey string
}

func NewJWTAuthenticator() *JWTAuthenticator {
	return &JWTAuthenticator{
		secretKey: os.Getenv("JWT_SECRET"), // Store securely
	}
}

func (j *JWTAuthenticator) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

func (j *JWTAuthenticator) Login(w http.ResponseWriter, r *http.Request) {
	userID := "12345" // Simulated user (replace with actual authentication)

	token, err := j.generateToken(userID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

func (j *JWTAuthenticator) Callback(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"message": "JWT authentication successful"}`))
}

func (j *JWTAuthenticator) Logout(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"message": "Logout not applicable for JWT"}`))
}

func (j *JWTAuthenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(j.secretKey), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims["user_id"])
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
