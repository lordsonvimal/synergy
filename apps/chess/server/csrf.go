package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName = "csrf_token"
	csrfFormField  = "_csrf"
	csrfTokenBytes = 32
)

func generateToken() (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isSameOrigin(r *http.Request) bool {
	host := r.Host

	origin := r.Header.Get("Origin")
	if origin != "" {
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return u.Host == host
	}

	referer := r.Header.Get("Referer")
	if referer != "" {
		u, err := url.Parse(referer)
		if err != nil {
			return false
		}
		return u.Host == host
	}

	return false
}

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(csrfCookieName)
		if err != nil || cookie == "" {
			token, genErr := generateToken()
			if genErr != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			cookie = token
			c.SetCookie(csrfCookieName, cookie, 0, "/", "", false, false)
		}

		c.Set(csrfCookieName, cookie)

		if c.Request.Method == http.MethodPost {
			formToken := c.PostForm(csrfFormField)
			if formToken != "" {
				if formToken != cookie {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"error": "CSRF validation failed",
					})
					return
				}
			} else {
				if !isSameOrigin(c.Request) {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"error": "CSRF validation failed",
					})
					return
				}
			}
		}

		c.Next()
	}
}

func CSRFToken(c *gin.Context) string {
	token, _ := c.Get(csrfCookieName)
	if s, ok := token.(string); ok {
		return s
	}
	return ""
}
