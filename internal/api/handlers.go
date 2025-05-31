package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/hackclub/geocoder/internal/cache"
	"github.com/hackclub/geocoder/internal/database"
	"github.com/hackclub/geocoder/internal/geocoding"
	"github.com/hackclub/geocoder/internal/geoip"
	"github.com/hackclub/geocoder/internal/middleware"
	"github.com/hackclub/geocoder/internal/models"
)

type Handlers struct {
	db            database.DatabaseInterface
	geocodeClient *geocoding.Client
	geoipClient   *geoip.Client
	cacheService  *cache.CacheService
	wsClients     map[*websocket.Conn]bool
	wsBroadcast   chan models.WebSocketMessage
	upgrader      websocket.Upgrader
}

func NewHandlers(db database.DatabaseInterface, geocodeClient *geocoding.Client, geoipClient *geoip.Client, cacheService *cache.CacheService) *Handlers {
	h := &Handlers{
		db:            db,
		geocodeClient: geocodeClient,
		geoipClient:   geoipClient,
		cacheService:  cacheService,
		wsClients:     make(map[*websocket.Conn]bool),
		wsBroadcast:   make(chan models.WebSocketMessage, 100),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
	}

	// Start WebSocket broadcast goroutine
	go h.handleWebSocketBroadcast()

	return h
}

// v1/geocode endpoint
func (h *Handlers) HandleGeocode(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	address := r.URL.Query().Get("address")
	if address == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_ADDRESS", "Address parameter is required")
		return
	}

	apiKey, ok := r.Context().Value(middleware.APIKeyContextKey).(*models.APIKey)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "INVALID_API_KEY", "API key required")
		return
	}
	
	// Check cache first
	cached, cacheHit := h.cacheService.GetGeocodeResult(address)
	var result *geocoding.GeocodeResponse
	var err error

	if cacheHit {
		result = cached
	} else {
		// Make external API call
		if !h.geocodeClient.IsConfigured() {
			h.writeErrorResponse(w, http.StatusServiceUnavailable, "EXTERNAL_API_ERROR", "Google Geocoding API not configured")
			return
		}

		result, err = h.geocodeClient.Geocode(address)
		if err != nil {
			h.writeErrorResponse(w, http.StatusBadGateway, "EXTERNAL_API_ERROR", fmt.Sprintf("Failed to geocode address: %v", err))
			return
		}

		// Cache the result
		_ = h.cacheService.SetGeocodeResult(address, result)
	}

	responseTime := int(time.Since(startTime).Milliseconds())

	// Log usage
	_ = h.db.LogUsage(apiKey.ID, "v1/geocode", cacheHit, responseTime)

	// Log activity
	apiSource := "cache"
	if !cacheHit {
		apiSource = "google"
	}
	_ = h.db.LogActivity(apiKey.Name, "v1/geocode", address, len(result.Results), responseTime, apiSource, cacheHit, r.RemoteAddr, r.UserAgent())
	
	// Broadcast activity update
	activity := &models.ActivityLog{
		Timestamp:      time.Now(),
		APIKeyName:     apiKey.Name,
		Endpoint:       "v1/geocode",
		QueryText:      address,
		ResultCount:    len(result.Results),
		ResponseTimeMs: responseTime,
		APISource:      apiSource,
		CacheHit:       cacheHit,
		IPAddress:      r.RemoteAddr,
		UserAgent:      r.UserAgent(),
	}
	h.broadcastActivity(activity)

	// Update cost tracking
	if !cacheHit {
		today := time.Now().Truncate(24 * time.Hour)
		_ = h.db.UpdateCostTracking(today, 1, 0, 0, 0, 0.005) // $0.005 per Google API call
	} else {
		today := time.Now().Truncate(24 * time.Hour)
		_ = h.db.UpdateCostTracking(today, 0, 1, 0, 0, 0)
	}

	// Send WebSocket update if there are results
	if len(result.Results) > 0 {
		location := result.Results[0].Geometry.Location
		h.broadcastUpdate(models.WebSocketMessage{
			Type:      "geocode_request",
			Lat:       location.Lat,
			Lng:       location.Lng,
			CacheHit:  cacheHit,
			Endpoint:  "v1/geocode",
			Address:   address,
			Timestamp: time.Now(),
		})
	}

	// Broadcast updated stats
	h.broadcastStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// v1/geoip endpoint
func (h *Handlers) HandleGeoIP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_IP", "IP parameter is required")
		return
	}

	// Validate IP address
	if net.ParseIP(ip) == nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_IP", "Invalid IP address format")
		return
	}

	apiKey, ok := r.Context().Value(middleware.APIKeyContextKey).(*models.APIKey)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "INVALID_API_KEY", "API key required")
		return
	}

	// Check cache first
	cached, cacheHit := h.cacheService.GetIPResult(ip)
	var result *geoip.IPInfoResponse
	var err error

	if cacheHit {
		result = cached
	} else {
		// Make external API call
		result, err = h.geoipClient.GetIPInfo(ip)
		if err != nil {
			h.writeErrorResponse(w, http.StatusBadGateway, "EXTERNAL_API_ERROR", fmt.Sprintf("Failed to get IP info: %v", err))
			return
		}

		// Cache the result
		_ = h.cacheService.SetIPResult(ip, result)
	}

	responseTime := int(time.Since(startTime).Milliseconds())

	// Log usage
	_ = h.db.LogUsage(apiKey.ID, "v1/geoip", cacheHit, responseTime)

	// Log activity
	apiSource := "cache"
	if !cacheHit {
		apiSource = "ipinfo"
	}
	resultCount := 0
	if result.City != "" || result.Region != "" || result.Country != "" {
		resultCount = 1
	}
	_ = h.db.LogActivity(apiKey.Name, "v1/geoip", ip, resultCount, responseTime, apiSource, cacheHit, r.RemoteAddr, r.UserAgent())
	
	// Broadcast activity update
	activity := &models.ActivityLog{
		Timestamp:      time.Now(),
		APIKeyName:     apiKey.Name,
		Endpoint:       "v1/geoip",
		QueryText:      ip,
		ResultCount:    resultCount,
		ResponseTimeMs: responseTime,
		APISource:      apiSource,
		CacheHit:       cacheHit,
		IPAddress:      r.RemoteAddr,
		UserAgent:      r.UserAgent(),
	}
	h.broadcastActivity(activity)

	// Update cost tracking
	if !cacheHit {
		today := time.Now().Truncate(24 * time.Hour)
		_ = h.db.UpdateCostTracking(today, 0, 0, 1, 0, 0.001) // $0.001 per IPinfo API call
	} else {
		today := time.Now().Truncate(24 * time.Hour)
		_ = h.db.UpdateCostTracking(today, 0, 0, 0, 1, 0)
	}

	// Send WebSocket update if location is available
	if result.Loc != "" {
		parts := strings.Split(result.Loc, ",")
		if len(parts) == 2 {
			if lat, err := strconv.ParseFloat(parts[0], 64); err == nil {
				if lng, err := strconv.ParseFloat(parts[1], 64); err == nil {
					h.broadcastUpdate(models.WebSocketMessage{
						Type:      "geoip_request",
						Lat:       lat,
						Lng:       lng,
						CacheHit:  cacheHit,
						Endpoint:  "v1/geoip",
						IP:        ip,
						Timestamp: time.Now(),
					})
				}
			}
		}
	}

	// Broadcast updated stats
	h.broadcastStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Health check endpoint
func (h *Handlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	status := models.HealthStatus{
		Timestamp:        time.Now(),
		DatabaseConnected: true,
		Services:         make(map[string]string),
	}

	// Check database connection
	if err := h.db.Ping(); err != nil {
		status.DatabaseConnected = false
		status.Services["database"] = "unhealthy"
		status.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		status.Services["database"] = "healthy"
	}

	// Check external APIs
	googleConfigured := h.geocodeClient.IsConfigured()
	ipinfoConfigured := h.geoipClient.IsConfigured()

	if googleConfigured {
		status.Services["google_geocoding"] = "healthy"
	} else {
		status.Services["google_geocoding"] = "not_configured"
	}

	if ipinfoConfigured {
		status.Services["ipinfo"] = "healthy"
	} else {
		status.Services["ipinfo"] = "not_configured"
	}

	// Determine overall status
	if status.Status == "" {
		if !googleConfigured && !ipinfoConfigured {
			status.Status = "unhealthy"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else if !googleConfigured || !ipinfoConfigured {
			status.Status = "degraded"
		} else {
			status.Status = "healthy"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Admin endpoints
func (h *Handlers) HandleAdminKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.handleGetAPIKeys(w, r)
	case "POST":
		h.handleCreateAPIKey(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) handleGetAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.db.GetAllAPIKeys()
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve API keys")
		return
	}

	// Remove key hashes from response
	for i := range keys {
		keys[i].KeyHash = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

func (h *Handlers) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON")
		return
	}

	if req.Name == "" || req.Prefix == "" || req.Owner == "" || req.AppName == "" || req.Environment == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Name, owner, app_name, environment, and prefix are required")
		return
	}

	if req.RateLimitPerSecond <= 0 {
		req.RateLimitPerSecond = 10
	}

	// Generate API key
	apiKey := h.generateAPIKey(req.Prefix)
	keyHash := database.HashAPIKey(apiKey)

	// Save to database
	dbKey, err := h.db.CreateAPIKey(keyHash, req.Name, req.Owner, req.AppName, req.Environment, req.RateLimitPerSecond)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create API key")
		return
	}

	response := models.CreateAPIKeyResponse{
		ID:                 dbKey.ID,
		Name:               dbKey.Name,
		Owner:              dbKey.Owner,
		AppName:            dbKey.AppName,
		Environment:        dbKey.Environment,
		Key:                apiKey,
		RateLimitPerSecond: dbKey.RateLimitPerSecond,
		CreatedAt:          dbKey.CreatedAt,
	}

	// Broadcast updated stats (new API key affects active count)
	h.broadcastStats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *Handlers) HandleUpdateAPIKeyRateLimit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["key_id"]

	var req models.UpdateRateLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON")
		return
	}

	if req.RateLimitPerSecond <= 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Rate limit must be positive")
		return
	}

	err := h.db.UpdateAPIKeyRateLimit(keyID, req.RateLimitPerSecond)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update rate limit")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) HandleDeactivateAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["key_id"]

	err := h.db.DeactivateAPIKey(keyID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to deactivate API key")
		return
	}

	// Broadcast updated stats (deactivated key affects active count)
	h.broadcastStats()

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) HandleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetStats()
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve stats")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handlers) HandleAdminActivity(w http.ResponseWriter, r *http.Request) {
	activities, err := h.db.GetRecentActivity()
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve activity log")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activities)
}

// WebSocket handling
func (h *Handlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	h.wsClients[conn] = true

	// Keep connection alive and handle disconnection
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			delete(h.wsClients, conn)
			break
		}
	}
}

func (h *Handlers) broadcastUpdate(message models.WebSocketMessage) {
	select {
	case h.wsBroadcast <- message:
	default:
		// Channel is full, skip this update
	}
}

func (h *Handlers) broadcastActivity(activity *models.ActivityLog) {
	message := models.WebSocketMessage{
		Type:      "activity_update",
		Timestamp: time.Now(),
		Activity:  activity,
	}

	select {
	case h.wsBroadcast <- message:
	default:
		// Channel is full, skip this update
	}
}

func (h *Handlers) broadcastStats() {
	stats, err := h.db.GetStats()
	if err != nil {
		return // Skip if we can't get stats
	}

	message := models.WebSocketMessage{
		Type:      "stats_update",
		Timestamp: time.Now(),
		Stats:     stats,
	}

	select {
	case h.wsBroadcast <- message:
	default:
		// Channel is full, skip this update
	}
}

func (h *Handlers) handleWebSocketBroadcast() {
	for message := range h.wsBroadcast {
		for client := range h.wsClients {
			err := client.WriteJSON(message)
			if err != nil {
				client.Close()
				delete(h.wsClients, client)
			}
		}
	}
}

func (h *Handlers) generateAPIKey(prefix string) string {
	// Simple random string generation (in production, use crypto/rand)
	return fmt.Sprintf("%s_live_sk_%d", prefix, time.Now().UnixNano()%1000000000)
}

func (h *Handlers) HandleUnsupportedVersion(w http.ResponseWriter, r *http.Request) {
	h.writeErrorResponse(w, http.StatusNotFound, "UNSUPPORTED_VERSION", "API version not supported. Use /v1/ prefix for API endpoints.")
}

func (h *Handlers) writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:      errorCode,
			Message:   message,
			Timestamp: time.Now(),
		},
	}

	json.NewEncoder(w).Encode(errorResp)
}
