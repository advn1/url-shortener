package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/advn1/url-shortener/internal/jsonutils"
	"github.com/advn1/url-shortener/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func AuthMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("user_cookie")

			createNewUser := func() (string, error) {
				newID := uuid.New()
				expiration := time.Now().Add(time.Hour * 24 * 100)

				tokenString, err := createToken(newID, expiration).SignedString([]byte(key))
				if err != nil {
					return "", err
				}

				setCookie(w, tokenString, expiration)
				return newID.String(), nil
			}

			if err != nil {
				uid, err := createNewUser()
				if err != nil {
					jsonutils.WriteInternalError(w)
					return
				}

				ctx := context.WithValue(r.Context(), models.UserIDKey, uid)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			claims := &models.UserClaims{}
			token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (any, error) {
				return []byte(key), nil
			})

			if err != nil || !token.Valid {
				uid, err := createNewUser()
				if err != nil {
					jsonutils.WriteInternalError(w)
					return
				}
				ctx := context.WithValue(r.Context(), models.UserIDKey, uid)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			ctx := context.WithValue(r.Context(), models.UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func createToken(id uuid.UUID, exp time.Time) *jwt.Token {
	// create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, models.UserClaims{
		UserID: id.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	})

	return token
}

func setCookie(w http.ResponseWriter, token string, exp time.Time) {
	// set cookie
	cookie := &http.Cookie{
		Name:     "user_cookie",
		Value:    token,
		Expires:  exp,
		HttpOnly: true,
		Secure:   false, // in dev ok
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
}
