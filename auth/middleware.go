package auth

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type JWTConfig struct {
	SECRET_KEY string
}

func LoadJWTConfig() *JWTConfig {
	return &JWTConfig{
		SECRET_KEY: getEnv("SECRET_KEY", ""),
	}
}

type contextKey string

const (
	ContextUserID   contextKey = "userID"
	ContextUserRole contextKey = "userRole"
)

func RoleMiddleware(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Call our new helper function
			userID, userRole, err := GetUserFromToken(r)
			if err != nil {
				http.Error(w, "Unauthorized: Please login", http.StatusUnauthorized)
				return
			}

			// Check if the user has the required role
			authorized := false
			for _, role := range requiredRoles {
				if userRole == role {
					authorized = true
					break
				}
			}

			if !authorized {
				http.Error(w, "Forbidden - Insufficient role", http.StatusForbidden)
				return
			}

			// Add to context so your Handlers can use them easily
			ctx := context.WithValue(r.Context(), ContextUserID, userID)
			ctx = context.WithValue(ctx, ContextUserRole, userRole)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getEnv(key, defaultValue string) string {
	_ = godotenv.Load()

	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetUserFromToken extracts the userID and userRole from the "token" cookie
func GetUserFromToken(r *http.Request) (string, string, error) {
	// 1. Get the cookie from the request
	cookie, err := r.Cookie("token")
	if err != nil {
		return "", "", err // Cookie not found
	}

	// 2. Verify the token string stored in the cookie
	_, claims, err := VerifyToken(cookie.Value)
	if err != nil {
		return "", "", err // Token invalid or expired
	}

	// 3. Extract data from claims
	userID := claims.Subject
	userRole := ""
	if len(claims.Audience) > 0 {
		userRole = claims.Audience[0]
	}

	return userID, userRole, nil
}
