package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
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
	cached, cacheHit := h.cacheService.GetStandardGeocodeResult(address)
	var result *models.GeocodeAPIResponse
	var err error

	if cacheHit {
		result = cached
	} else {
		// Make external API call
		if !h.geocodeClient.IsConfigured() {
			h.writeErrorResponse(w, http.StatusServiceUnavailable, "EXTERNAL_API_ERROR", "Google Geocoding API not configured")
			return
		}

		result, err = h.geocodeClient.GeocodeToStandardFormat(address)
		if err != nil {
			h.writeErrorResponse(w, http.StatusBadGateway, "EXTERNAL_API_ERROR", fmt.Sprintf("Failed to geocode address: %v", err))
			return
		}

		// Cache the result
		_ = h.cacheService.SetStandardGeocodeResult(address, result)
	}

	responseTime := int(time.Since(startTime).Milliseconds())

	// Log usage
	_ = h.db.LogUsage(apiKey.ID, "v1/geocode", cacheHit, responseTime)

	// Log activity
	apiSource := "cache"
	if !cacheHit {
		apiSource = "google"
	}
	resultCount := 1 // Standard format always returns 1 result when successful
	if result.Lat == 0 && result.Lng == 0 {
		resultCount = 0 // No valid coordinates found
	}
	_ = h.db.LogActivity(apiKey.Name, "v1/geocode", address, resultCount, responseTime, apiSource, cacheHit, extractIP(r.RemoteAddr), r.UserAgent())

	// Broadcast activity update
	activity := &models.ActivityLog{
		Timestamp:      time.Now(),
		APIKeyName:     apiKey.Name,
		Endpoint:       "v1/geocode",
		QueryText:      address,
		ResultCount:    resultCount,
		ResponseTimeMs: responseTime,
		APISource:      apiSource,
		CacheHit:       cacheHit,
		IPAddress:      extractIP(r.RemoteAddr),
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
	if result.Lat != 0 || result.Lng != 0 {
		h.broadcastUpdate(models.WebSocketMessage{
			Type:      "geocode_request",
			Lat:       result.Lat,
			Lng:       result.Lng,
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

// v1/geocode_structured endpoint
func (h *Handlers) HandleGeocodeStructured(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// Parse structured address from query parameters
	structuredAddr := models.StructuredAddress{
		AddressLine1: r.URL.Query().Get("address_line_1"),
		AddressLine2: r.URL.Query().Get("address_line_2"),
		City:         r.URL.Query().Get("city"),
		State:        r.URL.Query().Get("state"),
		PostalCode:   r.URL.Query().Get("postal_code"),
		Country:      r.URL.Query().Get("country"),
	}

	// Validate that at least one field is provided
	if structuredAddr.AddressLine1 == "" && structuredAddr.AddressLine2 == "" && structuredAddr.City == "" && structuredAddr.State == "" && structuredAddr.PostalCode == "" && structuredAddr.Country == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_ADDRESS", "At least one address field is required")
		return
	}

	apiKey, ok := r.Context().Value(middleware.APIKeyContextKey).(*models.APIKey)
	if !ok {
		h.writeErrorResponse(w, http.StatusUnauthorized, "INVALID_API_KEY", "API key required")
		return
	}

	// Convert structured address to formatted string for caching and geocoding
	address := structuredAddr.ToFormattedString()

	// Check cache first
	cached, cacheHit := h.cacheService.GetStandardGeocodeResult(address)
	var result *models.GeocodeAPIResponse
	var err error

	if cacheHit {
		result = cached
	} else {
		// Make external API call
		if !h.geocodeClient.IsConfigured() {
			h.writeErrorResponse(w, http.StatusServiceUnavailable, "EXTERNAL_API_ERROR", "Google Geocoding API not configured")
			return
		}

		result, err = h.geocodeClient.GeocodeToStandardFormat(address)
		if err != nil {
			h.writeErrorResponse(w, http.StatusBadGateway, "EXTERNAL_API_ERROR", fmt.Sprintf("Failed to geocode address: %v", err))
			return
		}

		// Cache the result
		_ = h.cacheService.SetStandardGeocodeResult(address, result)
	}

	responseTime := int(time.Since(startTime).Milliseconds())

	// Log usage
	_ = h.db.LogUsage(apiKey.ID, "v1/geocode_structured", cacheHit, responseTime)

	// Broadcast activity update
	activity := &models.ActivityLog{
		Timestamp:      time.Now(),
		APIKeyName:     apiKey.Name,
		Endpoint:       "v1/geocode_structured",
		QueryText:      address,
		ResultCount:    1,
		ResponseTimeMs: responseTime,
		APISource:      "google",
		CacheHit:       cacheHit,
		IPAddress:      extractIP(r.RemoteAddr),
		UserAgent:      r.UserAgent(),
	}
	if cacheHit {
		activity.APISource = "cache"
	}
	if result.Lat == 0 && result.Lng == 0 {
		activity.ResultCount = 0
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
	if result.Lat != 0 || result.Lng != 0 {
		h.broadcastUpdate(models.WebSocketMessage{
			Type:      "geocode_request",
			Lat:       result.Lat,
			Lng:       result.Lng,
			CacheHit:  cacheHit,
			Endpoint:  "v1/geocode_structured",
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
	cached, cacheHit := h.cacheService.GetStandardIPResult(ip)
	var result *models.GeoIPAPIResponse
	var err error

	if cacheHit {
		result = cached
	} else {
		// Make external API call
		result, err = h.geoipClient.GetIPInfoToStandardFormat(ip)
		if err != nil {
			h.writeErrorResponse(w, http.StatusBadGateway, "EXTERNAL_API_ERROR", fmt.Sprintf("Failed to get IP info: %v", err))
			return
		}

		// Cache the result
		_ = h.cacheService.SetStandardIPResult(ip, result)
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
	if result.City != "" || result.Region != "" || result.CountryCode != "" {
		resultCount = 1
	}
	_ = h.db.LogActivity(apiKey.Name, "v1/geoip", ip, resultCount, responseTime, apiSource, cacheHit, extractIP(r.RemoteAddr), r.UserAgent())

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
		IPAddress:      extractIP(r.RemoteAddr),
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
	if result.Lat != 0 || result.Lng != 0 {
		h.broadcastUpdate(models.WebSocketMessage{
			Type:      "geoip_request",
			Lat:       result.Lat,
			Lng:       result.Lng,
			CacheHit:  cacheHit,
			Endpoint:  "v1/geoip",
			IP:        ip,
			Timestamp: time.Now(),
		})
	}

	// Broadcast updated stats
	h.broadcastStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Health check endpoint
func (h *Handlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	status := models.HealthStatus{
		Timestamp:         time.Now(),
		DatabaseConnected: true,
		Services:          make(map[string]string),
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

	if req.Name == "" || req.Owner == "" || req.AppName == "" || req.Environment == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Name, owner, app_name, and environment are required")
		return
	}

	if req.RateLimitPerSecond <= 0 {
		req.RateLimitPerSecond = 10
	}

	// Auto-generate prefix from user input
	prefix := fmt.Sprintf("%s_%s_%s", req.Owner, req.AppName, req.Environment)

	// Generate API key
	apiKey := h.generateAPIKey(prefix)
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

func (h *Handlers) HandleUsageSummary(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for pagination
	page := 1
	pageSize := 10

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	summary, err := h.db.GetAPIKeyUsageSummary(page, pageSize)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve usage summary")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
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
	// Generate 20 random bytes for secure key
	randomBytes := make([]byte, 20)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fallback to timestamp if crypto/rand fails
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano()%1000000000)
	}

	randomString := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%s_%s", prefix, randomString)
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

// extractIP extracts the IP address from a RemoteAddr string (removes port if present)
func extractIP(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr // fallback if not in host:port format
}
