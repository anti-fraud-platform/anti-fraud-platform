package main

import (
	"anti-fraud/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID     int    `json:"user_id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	CampaignID string `json:"campaign_id"`
	jwt.RegisteredClaims
}

type AttackLog struct {
	ID          int64     `json:"id"`
	IP          string    `json:"ip"`
	CampaignID  string    `json:"campaign_id"`
	UserAgent   string    `json:"user_agent"`
	Reason      string    `json:"reason"`
	ProcessedAt time.Time `json:"processed_at"`
}

type contextKey string

const (
	contextKeyUserID     contextKey = "user_id"
	contextKeyUsername   contextKey = "username"
	contextKeyRole       contextKey = "role"
	contextKeyCampaignID contextKey = "campaign_id"
)

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Username == "" || req.Password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// bcrypt hashing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if req.CampaignID == "" {
		req.CampaignID = "default_campaign"
	}

	query := `
		INSERT INTO users (email, username, password_hash, role, campaign_id) 
		VALUES ($1, $2, $3, 'user', $4) 
		RETURNING id
	`
	var userID int
	err = db.QueryRow(query, req.Email, req.Username, string(hashedPassword), req.CampaignID).Scan(&userID)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			http.Error(w, "Email or username already exists", http.StatusConflict)
			return
		}

		log.Printf("Error inserting user into DB: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User registered successfully",
		"user_id": userID,
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var user models.User
	query := `SELECT id, email, username, password_hash, role, campaign_id FROM users WHERE email = $1`
	err := db.QueryRow(query, req.Email).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.Role, &user.CampaignID,
	)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// token generation 24h liveness
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:     user.ID,
		Username:   user.Username,
		Role:       user.Role,
		CampaignID: user.CampaignID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		log.Printf("Error signing JWT: %v", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}

// authMiddleware checks JWT token and adds user's credentials to the ctx
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// take token from header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		// "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, contextKeyUsername, claims.Username)
		ctx = context.WithValue(ctx, contextKeyRole, claims.Role)
		ctx = context.WithValue(ctx, contextKeyCampaignID, claims.CampaignID)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// requireRole creates middleware for checking user's role
func requireRole(requiredRole string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(contextKeyRole).(string)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if role != requiredRole {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// handleMe returns info about user from JWT
func handleMe(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(contextKeyUserID).(int)
	username := r.Context().Value(contextKeyUsername).(string)
	role := r.Context().Value(contextKeyRole).(string)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":  userID,
		"username": username,
		"role":     role,
	})
}

// handleAdminUsers only for admins
func handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	// plug here
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Admin endpoint: list of all users would be here",
	})
}

// handleAttacks return list of attacks
func handleAttacks(w http.ResponseWriter, r *http.Request) {
	//take data from ctx (Middlware)
	role := r.Context().Value(contextKeyRole).(string)
	campaignID := r.Context().Value(contextKeyCampaignID).(string)

	var query string
	var args []interface{}

	if role == "admin" {
		query = `
			SELECT id, ip, campaign_id, user_agent, reason, processed_at 
			FROM click_logs 
			WHERE is_bot = TRUE OR reason != 'allowed' 
			ORDER BY processed_at DESC 
			LIMIT 100
		`
	} else {
		query = `
			SELECT id, ip, campaign_id, user_agent, reason, processed_at 
			FROM click_logs 
			WHERE campaign_id = $1 AND (is_bot = TRUE OR reason != 'allowed') 
			ORDER BY processed_at DESC 
			LIMIT 100
		`
		args = append(args, campaignID)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying attacks: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var attacks []AttackLog
	for rows.Next() {
		var a AttackLog
		err := rows.Scan(&a.ID, &a.IP, &a.CampaignID, &a.UserAgent, &a.Reason, &a.ProcessedAt)
		if err != nil {
			log.Printf("Error scanning attack row: %v", err)
			continue
		}
		attacks = append(attacks, a)
	}

	if attacks == nil {
		attacks = []AttackLog{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attacks)
}
