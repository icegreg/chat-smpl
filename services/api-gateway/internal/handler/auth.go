package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/centrifugo"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/middleware"
)

type AuthHandler struct {
	usersServiceURL  string
	centrifugoClient *centrifugo.Client
	httpClient       *http.Client
	log              logger.Logger
}

func NewAuthHandler(usersServiceURL string, centrifugoClient *centrifugo.Client, log logger.Logger) *AuthHandler {
	return &AuthHandler{
		usersServiceURL:  usersServiceURL,
		centrifugoClient: centrifugoClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: log,
	}
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public routes (no auth required)
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.RefreshToken)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Post("/logout", h.Logout)
		r.Get("/me", h.GetCurrentUser)
		r.Put("/me", h.UpdateCurrentUser)
		r.Put("/me/password", h.ChangePassword)
	})

	return r
}

// Register proxies registration request to users service
// POST /api/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	h.proxyToUsers(w, r, "/api/auth/register")
}

// Login proxies login request to users service
// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.proxyToUsers(w, r, "/api/auth/login")
}

// RefreshToken proxies refresh request to users service
// POST /api/auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	h.proxyToUsers(w, r, "/api/auth/refresh")
}

// Logout proxies logout request to users service
// POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.proxyToUsers(w, r, "/api/auth/logout")
}

// GetCurrentUser returns current user info
// GET /api/auth/me
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Proxy to users service with user ID
	targetURL := fmt.Sprintf("%s/api/users/%s", h.usersServiceURL, userID.String())

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create request")
		return
	}

	// Copy authorization header
	req.Header.Set("Authorization", r.Header.Get("Authorization"))

	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.log.Error("failed to proxy request", "error", err)
		h.respondError(w, http.StatusBadGateway, "service unavailable")
		return
	}
	defer resp.Body.Close()

	// Copy response
	h.copyResponse(w, resp)
}

// UpdateCurrentUser updates current user info
// PUT /api/auth/me
func (h *AuthHandler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	// Proxy to users service with user ID
	targetURL := fmt.Sprintf("%s/api/users/%s", h.usersServiceURL, userID.String())

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPut, targetURL, bytes.NewReader(body))
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create request")
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", r.Header.Get("Authorization"))

	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.log.Error("failed to proxy request", "error", err)
		h.respondError(w, http.StatusBadGateway, "service unavailable")
		return
	}
	defer resp.Body.Close()

	h.copyResponse(w, resp)
}

// ChangePassword changes user password
// PUT /api/auth/me/password
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	// Proxy to users service
	targetURL := fmt.Sprintf("%s/api/users/%s/password", h.usersServiceURL, userID.String())

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPut, targetURL, bytes.NewReader(body))
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create request")
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", r.Header.Get("Authorization"))

	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.log.Error("failed to proxy request", "error", err)
		h.respondError(w, http.StatusBadGateway, "service unavailable")
		return
	}
	defer resp.Body.Close()

	h.copyResponse(w, resp)
}

func (h *AuthHandler) proxyToUsers(w http.ResponseWriter, r *http.Request, path string) {
	// Read original body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	targetURL := h.usersServiceURL + path

	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to create request")
		return
	}

	// Copy headers
	req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	if auth := r.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.log.Error("failed to proxy request", "error", err, "url", targetURL)
		h.respondError(w, http.StatusBadGateway, "service unavailable")
		return
	}
	defer resp.Body.Close()

	h.copyResponse(w, resp)
}

func (h *AuthHandler) copyResponse(w http.ResponseWriter, resp *http.Response) {
	// Copy headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	// Copy body
	io.Copy(w, resp.Body)
}

func (h *AuthHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *AuthHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

// CentrifugoHandler handles Centrifugo token generation
type CentrifugoHandler struct {
	centrifugoClient *centrifugo.Client
	log              logger.Logger
}

func NewCentrifugoHandler(centrifugoClient *centrifugo.Client, log logger.Logger) *CentrifugoHandler {
	return &CentrifugoHandler{
		centrifugoClient: centrifugoClient,
		log:              log,
	}
}

func (h *CentrifugoHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/connection-token", h.GetConnectionToken)
	r.Post("/subscription-token", h.GetSubscriptionToken)

	return r
}

// GetConnectionToken returns a connection token for Centrifugo
// GET /api/centrifugo/connection-token
func (h *CentrifugoHandler) GetConnectionToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Token expires in 1 hour
	exp := time.Now().Add(time.Hour).Unix()
	token := h.centrifugoClient.GenerateConnectionToken(userID.String(), exp)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"expires_at": exp,
	})
}

// GetSubscriptionToken returns a subscription token for a channel
// POST /api/centrifugo/subscription-token
func (h *CentrifugoHandler) GetSubscriptionToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Channel string `json:"channel"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Channel == "" {
		h.respondError(w, http.StatusBadRequest, "channel required")
		return
	}

	// Token expires in 1 hour
	exp := time.Now().Add(time.Hour).Unix()
	token := h.centrifugoClient.GenerateSubscriptionToken(userID.String(), req.Channel, exp)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"channel":    req.Channel,
		"expires_at": exp,
	})
}

func (h *CentrifugoHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *CentrifugoHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
