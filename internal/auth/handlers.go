package auth

import (
	"encoding/json"
	"log"
	"net/http"
)

// AuthHandlers groups the HTTP handlers for register, login, and me endpoints.
type AuthHandlers struct {
	store *UserStore
}

// NewAuthHandlers creates an instance backed by the given UserStore.
func NewAuthHandlers(store *UserStore) *AuthHandlers {
	return &AuthHandlers{store: store}
}

// SeedAdmin creates a default admin user if one does not already exist.
// Intended to be called once at startup. Errors are logged but do not
// prevent the service from starting — the admin can be created later.
func (h *AuthHandlers) SeedAdmin(username, password string) {
	if err := h.store.CreateUser(username, password, "admin"); err != nil {
		log.Printf("SeedAdmin: user %q may already exist or DB error: %v", username, err)
	} else {
		log.Printf("SeedAdmin: created admin user %q", username)
	}
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

type userResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// RegisterHandler creates a new user. The request body must contain
// "username" and "password" fields. Default role is "viewer".
func (h *AuthHandlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"username and password are required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Username) < 3 || len(req.Username) > 64 {
		http.Error(w, `{"error":"username must be 3-64 characters"}`, http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		http.Error(w, `{"error":"password must be at least 6 characters"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.CreateUser(req.Username, req.Password, "viewer"); err != nil {
		// Most likely a unique-constraint violation on username.
		http.Error(w, `{"error":"username already exists"}`, http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "user created"})
}

// LoginHandler verifies credentials and returns a JWT token.
func (h *AuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"username and password are required"}`, http.StatusBadRequest)
		return
	}

	user, err := h.store.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	if !CheckPassword(user.PasswordHash, req.Password) {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token, err := GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		log.Printf("LoginHandler: failed to generate token: %v", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResponse{Token: token})
}

// MeHandler returns the profile of the currently authenticated user.
// Must be wrapped with RequireAuth middleware.
func (h *AuthHandlers) MeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	claims, ok := UserFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userResponse{
		ID:       claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	})
}
