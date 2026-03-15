package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env definitions
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it")
	}

	InitDB()

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "up and running"})
	})

	// Auth routes (public)
	mux.HandleFunc("POST /api/auth/register", csrfSkip(Register))
	mux.HandleFunc("POST /api/auth/verify-email", csrfSkip(VerifyOTP))
	mux.HandleFunc("POST /api/auth/login", csrfSkip(Login))
	mux.HandleFunc("POST /api/auth/refresh", csrfSkip(RefreshToken))
	mux.HandleFunc("POST /api/auth/forgot-password", csrfSkip(ForgotPassword))
	mux.HandleFunc("POST /api/auth/reset-password", csrfSkip(ResetPassword))

	// 2FA routes (authenticated)
	mux.HandleFunc("GET /api/auth/2fa/status", requireAuth(Get2FAStatus))
	mux.HandleFunc("POST /api/auth/2fa/setup", requireAuth(Setup2FA))
	mux.HandleFunc("POST /api/auth/2fa/verify-setup", requireAuth(VerifySetup2FA))
	mux.HandleFunc("POST /api/auth/2fa/disable", requireAuth(Disable2FA))

	// Wrap everything with CORS
	handler := corsMiddleware(mux)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Println("Server running on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// corsMiddleware handles CORS for all requests
func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:3000": true,
		"http://localhost:3001": true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-CSRF-Token")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight OPTIONS
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// csrfSkip wraps a handler, skipping CSRF for public auth routes
func csrfSkip(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Generate CSRF token on GET requests
		if r.Method == http.MethodGet {
			cookie, _ := r.Cookie("csrf_token")
			if cookie == nil {
				token, _ := GenerateRandomString(32)
				http.SetCookie(w, &http.Cookie{
					Name:     "csrf_token",
					Value:    token,
					Path:     "/",
					MaxAge:   3600,
					HttpOnly: false,
				})
			}
		}
		handler(w, r)
	}
}

// requireAuth middleware validates JWT and passes userID via request header
func requireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) < 8 || !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "Authorization header missing or invalid")
			return
		}

		tokenStr := authHeader[7:]
		claims, err := ValidateToken(tokenStr, false)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Invalid access token")
			return
		}

		// Store user info in request headers (simple approach for net/http)
		r.Header.Set("X-User-ID", formatUint(claims.UserID))
		r.Header.Set("X-User-Email", claims.Email)

		handler(w, r)
	}
}
