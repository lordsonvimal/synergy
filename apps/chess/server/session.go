package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lordsonvimal/synergy/apps/chess/config"
)

const playCookieName = "play_session"

// PlaySessionClaims is the payload stored in the play_session JWT cookie.
type PlaySessionClaims struct {
	GameID string `json:"game_id"`
	Role   string `json:"role"` // "white" | "black" | "spectator"
	jwt.RegisteredClaims
}

func signPlaySession(claims PlaySessionClaims) (string, error) {
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.SessionSecret())
}

func verifyPlaySession(tokenStr string) (PlaySessionClaims, error) {
	var claims PlaySessionClaims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return config.SessionSecret(), nil
	})
	if err != nil || !token.Valid {
		return PlaySessionClaims{}, errors.New("invalid session")
	}
	return claims, nil
}

// SetPlaySession writes a signed JWT into the play_session cookie.
func SetPlaySession(c *gin.Context, gameID, role string) error {
	tokenStr, err := signPlaySession(PlaySessionClaims{GameID: gameID, Role: role})
	if err != nil {
		return err
	}
	c.SetCookie(playCookieName, tokenStr, 7*24*3600, "/", "", false, true)
	return nil
}

// GetPlaySession reads and verifies the play_session cookie from the request.
func GetPlaySession(r *http.Request) (PlaySessionClaims, error) {
	cookie, err := r.Cookie(playCookieName)
	if err != nil {
		return PlaySessionClaims{}, err
	}
	return verifyPlaySession(cookie.Value)
}

// ClearPlaySession deletes the play_session cookie.
func ClearPlaySession(c *gin.Context) {
	c.SetCookie(playCookieName, "", -1, "/", "", false, true)
}
