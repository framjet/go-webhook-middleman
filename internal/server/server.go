package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	configApi "github.com/framjet/go-webhook-middleman/internal/config"
	metricsApi "github.com/framjet/go-webhook-middleman/internal/metrics"
	"github.com/framjet/go-webhook-middleman/internal/templateRenderer"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type WebhookServer struct {
	Config  *configApi.Config
	client  *http.Client
	logger  *slog.Logger
	metrics *metricsApi.Metrics
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

func NewWebhookServer(configPath string, timeout time.Duration, logger *slog.Logger) (*WebhookServer, error) {
	config, err := configApi.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	metrics := metricsApi.NewMetrics()

	return &WebhookServer{
		Config:  config,
		client:  client,
		logger:  logger,
		metrics: metrics,
	}, nil
}

func (ws *WebhookServer) findMatchingRoute(path, method string) *configApi.Route {
	for _, route := range ws.Config.Routes {
		if ws.routeMatches(route, path, method) {
			return &route
		}
	}
	return nil
}

func (ws *WebhookServer) routeMatches(route configApi.Route, path, method string) bool {
	// Check method
	methods := route.Methods
	if route.Method != "" {
		methods = append(methods, route.Method)
	}
	if len(methods) == 0 {
		methods = []string{"POST"} // default
	}

	methodMatches := false
	for _, m := range methods {
		if strings.ToUpper(m) == strings.ToUpper(method) {
			methodMatches = true
			break
		}
	}
	if !methodMatches {
		return false
	}

	// Check the path
	paths := route.Paths
	if route.Path != "" {
		paths = append(paths, route.Path)
	}

	for _, p := range paths {
		if ws.pathMatches(p, path) {
			return true
		}
	}

	return len(paths) == 0 // If no paths specified, match all
}

func (ws *WebhookServer) pathMatches(routePath, requestPath string) bool {
	// Simple path matching - could be enhanced with more sophisticated routing
	if routePath == requestPath {
		return true
	}

	// Basic parameter matching (this is simplified - gorilla/mux does the real work)
	routeParts := strings.Split(routePath, "/")
	requestParts := strings.Split(requestPath, "/")

	if len(routeParts) != len(requestParts) {
		return false
	}

	for i, part := range routeParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue // Parameter, matches anything
		}
		if part != requestParts[i] {
			return false
		}
	}

	return true
}

func (ws *WebhookServer) extractParams(routePath, requestPath string) map[string]string {
	params := make(map[string]string)

	routeParts := strings.Split(routePath, "/")
	requestParts := strings.Split(requestPath, "/")

	if len(routeParts) != len(requestParts) {
		return params
	}

	for i, part := range routeParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := part[1 : len(part)-1]
			params[paramName] = requestParts[i]
		}
	}

	return params
}

func (ws *WebhookServer) handleDynamicWebhook(w http.ResponseWriter, r *http.Request) {
	if err := ws.validateRequest(r); err != nil {
		ws.logger.Error("Request validation failed", "error", err, "remote_addr", r.RemoteAddr)
		ws.metrics.WebhooksProcessed.WithLabelValues("validation_error").Inc()
		ws.writeErrorResponse(w, http.StatusRequestEntityTooLarge, err.Error(), "REQUEST_TOO_LARGE")
		return
	}

	// Generate request ID for tracing
	requestID := uuid.New().String()
	logger := ws.logger.With("request_id", requestID)

	ctx := r.Context()
	start := time.Now()
	ws.metrics.WebhooksReceived.Inc()

	// Find matching route
	route := ws.findMatchingRoute(r.URL.Path, r.Method)
	if route == nil {
		logger.Warn("No matching route found",
			"path", r.URL.Path,
			"method", r.Method,
			"remote_addr", r.RemoteAddr)
		ws.metrics.WebhooksProcessed.WithLabelValues("no_route").Inc()
		ws.writeErrorResponse(w, http.StatusNotFound, "No matching route found", "NO_ROUTE_MATCH")
		return
	}

	// Extract parameters - use gorilla/mux vars if available, otherwise extract manually
	params := mux.Vars(r)
	if len(params) == 0 {
		// Fallback to manual extraction for the first matching path
		routePath := route.Path
		if len(route.Paths) > 0 {
			routePath = route.Paths[0]
		}
		params = ws.extractParams(routePath, r.URL.Path)
	}

	// Record route match
	ws.metrics.RoutesMatched.WithLabelValues(r.Method, r.URL.Path).Inc()

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read request body", "error", err, "params", params)
		ws.metrics.WebhooksProcessed.WithLabelValues("error").Inc()
		ws.writeErrorResponse(w, http.StatusBadRequest, "Failed to read request body", "BODY_READ_ERROR")
		return
	}

	logger.Debug("Received webhook",
		"params", params,
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"),
		"content_type", r.Header.Get("Content-Type"),
		"content_length", r.ContentLength,
		"body", string(body),
	)

	// Create template context
	templateCtx := templateRenderer.TemplateContext{
		Params:    params,
		Variables: ws.Config.Variables,
		Body:      string(body),
		Route:     *route,
		Request:   *r,
	}

	// Find matching destinations
	destinations := ws.findMatchingDestinations(route, params, templateCtx, r, string(body), logger)
	if len(destinations) == 0 {
		logger.Warn("No matching destinations found", "params", params)
		ws.metrics.WebhooksProcessed.WithLabelValues("no_destinations").Inc()
		ws.writeErrorResponse(w, http.StatusNotFound, "No matching destinations found", "NO_DESTINATIONS")
		return
	}

	logger.Info("Processing webhook",
		"params", params,
		"destinations", len(destinations),
		"body_size", len(body))

	// Forward to all matching destinations
	results := ws.forwardToDestinations(ctx, destinations, r.Header, logger)

	// Count successful forwards
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	duration := time.Since(start)
	logger.Info("Webhook processed",
		"params", params,
		"total_destinations", len(destinations),
		"successful", successCount,
		"duration", duration)

	// Record processing status
	if successCount == len(destinations) {
		ws.metrics.WebhooksProcessed.WithLabelValues("success").Inc()
	} else if successCount > 0 {
		ws.metrics.WebhooksProcessed.WithLabelValues("partial").Inc()
	} else {
		ws.metrics.WebhooksProcessed.WithLabelValues("failed").Inc()
	}

	responseData := &ResponseData{
		Destinations: destinations,
		SuccessCount: successCount,
		Duration:     duration,
		Results:      results,
		Params:       templateCtx.Params,
		Variables:    templateCtx.Variables,
		Body:         templateCtx.Body,
		Request:      r,
		ForwardedTo:  len(destinations),
		DurationMs:   duration.Milliseconds(),
		Successful:   successCount,
	}

	handler := NewResponseHandler(route, responseData)
	if err := handler.SendResponse(w); err != nil {
		// Log error and send fallback response
		ws.writeErrorResponse(w, http.StatusInternalServerError, "Failed to generate response", "RESPONSE_ERROR")
		return
	}
}

func (ws *WebhookServer) findMatchingDestinations(route *configApi.Route, params map[string]string, ctx templateRenderer.TemplateContext, request *http.Request, body string, logger *slog.Logger) []configApi.ResolvedDestination {
	var destinations []configApi.ResolvedDestination

	// Process matchers
	for _, matcher := range route.Matchers {
		if ws.matcherMatches(route, matcher, params, request, body, logger) {
			for _, destRef := range matcher.To {
				resolved, err := ws.resolveDestination(destRef, ctx)
				if err != nil {
					logger.Error("Failed to resolve destination", "error", err, "dest", destRef)
					continue
				}
				destinations = append(destinations, resolved)
			}
		}
	}

	return destinations
}

func (ws *WebhookServer) matcherMatches(route *configApi.Route, matcher *configApi.Matcher, params map[string]string, request *http.Request, body string, logger *slog.Logger) bool {
	userInfo := ""
	if request.URL.User != nil {
		userInfo = request.URL.User.String()
	}

	env := &configApi.MatcherEnv{
		Params:  params,
		Var:     ws.Config.Variables,
		Matcher: *matcher,
		Config:  *ws.Config,
		Route:   *route,
		Request: configApi.RequestData{
			Method: request.Method,
			Url: configApi.RequestUrlData{
				Full:     request.URL.String(),
				Scheme:   request.URL.Scheme,
				Host:     request.URL.Host,
				Path:     request.URL.Path,
				Query:    request.URL.Query(),
				Opaque:   request.URL.Opaque,
				Fragment: request.URL.Fragment,
				UserInfo: userInfo,
				RawQuery: request.URL.RawQuery,
			},
			Headers:     request.Header,
			Host:        request.Host,
			Body:        body,
			ContentType: request.Header.Get("content-type"),
			UserAgent:   request.UserAgent(),
			RemoteAddr:  request.RemoteAddr,
		},
	}

	result, err := matcher.Evaluate(env)
	if err != nil {
		logger.Warn("Matcher evaluation failed", "matcher", matcher, "error", err)

		return false
	}

	return result
}

func (ws *WebhookServer) valueMatches(routeValue interface{}, paramValue string, logger *slog.Logger) bool {
	switch v := routeValue.(type) {
	case string:
		return ws.stringMatches(v, paramValue, logger)
	case []interface{}:
		// List match - check each item in the array
		for _, item := range v {
			if str, ok := item.(string); ok && ws.stringMatches(str, paramValue, logger) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (ws *WebhookServer) stringMatches(pattern, value string, logger *slog.Logger) bool {
	// Check if the pattern is a regex format: /.../
	if len(pattern) >= 2 && pattern[0] == '/' && pattern[len(pattern)-1] == '/' {
		// Extract regex pattern between slashes
		regexPattern := pattern[1 : len(pattern)-1]
		matched, err := regexp.MatchString(regexPattern, value)
		if err != nil {
			logger.Error("Invalid regex pattern", "pattern", pattern, "error", err)
			return false
		}
		return matched
	}

	// Exact string match
	return pattern == value
}

func (ws *WebhookServer) resolveDestination(ref configApi.DestinationRef, ctx templateRenderer.TemplateContext) (configApi.ResolvedDestination, error) {
	resolved := configApi.ResolvedDestination{
		Method:  "POST", // default
		Headers: make(map[string]string),
		Body:    []byte(ctx.Body),
	}

	if ref.Name != "" {
		// Look up in global destinations
		if globalDest, exists := ws.Config.Destinations[ref.Name]; exists {
			var err error
			resolved.URL, err = templateRenderer.RenderTemplate(globalDest.URL, ctx)
			if err != nil {
				return resolved, fmt.Errorf("failed to render global destination URL: %w", err)
			}
			resolved.Name = ref.Name

			if globalDest.Method != "" {
				resolved.Method = globalDest.Method
			}
			if globalDest.Body != "" {
				bodyStr, err := templateRenderer.RenderTemplate(globalDest.Body, ctx)
				if err != nil {
					return resolved, fmt.Errorf("failed to render global destination body: %w", err)
				}
				resolved.Body = []byte(bodyStr)
			}
		} else {
			return resolved, fmt.Errorf("destination '%s' not found in config", ref.Name)
		}
	}

	// Override with inline destination settings
	if ref.URL != "" {
		var err error
		resolved.URL, err = templateRenderer.RenderTemplate(ref.URL, ctx)
		if err != nil {
			return resolved, fmt.Errorf("failed to render inline destination URL: %w", err)
		}
		resolved.Name = "inline"
	}
	if ref.Method != "" {
		resolved.Method = ref.Method
	}

	if ref.Headers != nil {
		for key, value := range ref.Headers {
			renderedValue, err := templateRenderer.RenderTemplate(value, ctx)
			if err != nil {
				return resolved, fmt.Errorf("failed to render header '%s': %w", key, err)
			}
			resolved.Headers[key] = renderedValue
		}
	}

	if ref.Body != "" {
		bodyStr, err := templateRenderer.RenderTemplate(ref.Body, ctx)
		if err != nil {
			return resolved, fmt.Errorf("failed to render inline destination body: %w", err)
		}
		resolved.Body = []byte(bodyStr)
	}

	if resolved.URL == "" {
		return resolved, fmt.Errorf("destination URL is empty after resolution")
	}

	return resolved, nil
}

func (ws *WebhookServer) forwardToDestinations(ctx context.Context, destinations []configApi.ResolvedDestination, headers http.Header, logger *slog.Logger) []configApi.ForwardResult {
	var wg sync.WaitGroup
	results := make([]configApi.ForwardResult, len(destinations))

	for i, dest := range destinations {
		wg.Add(1)
		go func(index int, destination configApi.ResolvedDestination) {
			defer wg.Done()
			results[index] = ws.forwardToDestination(ctx, destination, headers, logger)
		}(i, dest)
	}

	wg.Wait()
	return results
}

func (ws *WebhookServer) forwardToDestination(ctx context.Context, dest configApi.ResolvedDestination, headers http.Header, logger *slog.Logger) configApi.ForwardResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, dest.Method, dest.URL, bytes.NewReader(dest.Body))
	if err != nil {
		logger.Error("Failed to create request", "destination", dest.Name, "url", dest.URL, "error", err)
		duration := time.Since(start)
		ws.metrics.ForwardingTotal.WithLabelValues(dest.Name, "request_error").Inc()
		ws.metrics.ForwardingDuration.WithLabelValues(dest.Name, "request_error").Observe(duration.Seconds())
		return configApi.ForwardResult{
			Destination: dest.Name,
			URL:         dest.URL,
			Method:      dest.Method,
			Success:     false,
			Error:       fmt.Sprintf("failed to create request: %v", err),
			Duration:    duration.Milliseconds(),
		}
	}

	// Copy relevant headers
	for name, values := range headers {
		// Skip hop-by-hop headers
		if strings.ToLower(name) == "content-length" ||
			strings.ToLower(name) == "host" {
			continue
		}
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	for name, value := range dest.Headers {
		if value == "" {
			continue // Skip empty headers
		}
		req.Header.Set(name, value)
	}

	logger.Debug("Forwarding request",
		"destination", dest.Name,
		"url", dest.URL,
		"method", dest.Method,
		"headers", req.Header,
		"body_size", len(dest.Body),
		"body", string(dest.Body),
	)

	resp, err := ws.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Failed to forward request",
			"destination", dest.Name,
			"url", dest.URL,
			"method", dest.Method,
			"headers", dest.Headers,
			"error", err,
			"duration", duration)
		ws.metrics.ForwardingTotal.WithLabelValues(dest.Name, "network_error").Inc()
		ws.metrics.ForwardingDuration.WithLabelValues(dest.Name, "network_error").Observe(duration.Seconds())
		return configApi.ForwardResult{
			Destination: dest.Name,
			URL:         dest.URL,
			Method:      dest.Method,
			Headers:     dest.Headers,
			Success:     false,
			Error:       fmt.Sprintf("request failed: %v", err),
			Duration:    duration.Milliseconds(),
		}
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	status := "success"
	if !success {
		status = "http_error"
	}

	ws.metrics.ForwardingTotal.WithLabelValues(dest.Name, status).Inc()
	ws.metrics.ForwardingDuration.WithLabelValues(dest.Name, status).Observe(duration.Seconds())

	if logger.Enabled(ctx, slog.LevelDebug) {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Failed to read response body", "destination", dest.Name, "url", dest.URL, "error", err)
		} else {
			logger.Debug("Response body",
				"destination", dest.Name,
				"url", dest.URL,
				"method", dest.Method,
				"status", resp.StatusCode,
				"body", string(bodyBytes))
		}
	}

	defer resp.Body.Close()

	if success {
		logger.Debug("Successfully forwarded request",
			"destination", dest.Name,
			"url", dest.URL,
			"method", dest.Method,
			"headers", dest.Headers,
			"status", resp.Status,
			"statusCode", resp.StatusCode,
			"duration", duration)
	} else {
		logger.Warn("Request forwarded but received error status",
			"destination", dest.Name,
			"url", dest.URL,
			"method", dest.Method,
			"headers", dest.Headers,
			"status", resp.Status,
			"statusCode", resp.StatusCode,
			"duration", duration)
	}

	return configApi.ForwardResult{
		Destination: dest.Name,
		URL:         dest.URL,
		Method:      dest.Method,
		Headers:     dest.Headers,
		Success:     success,
		StatusCode:  resp.StatusCode,
		Duration:    duration.Milliseconds(),
	}
}

func (ws *WebhookServer) SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// Health check endpoint - GET only
	r.HandleFunc("/health", ws.healthCheck).Methods("GET")

	// Metrics endpoint - GET only
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Dynamic webhook routes
	for _, route := range ws.Config.Routes {
		paths := route.Paths
		if route.Path != "" {
			paths = append(paths, route.Path)
		}

		methods := route.Methods
		if route.Method != "" {
			methods = append(methods, route.Method)
		}
		if len(methods) == 0 {
			methods = []string{"POST"} // default
		}

		for _, path := range paths {
			r.HandleFunc(path, ws.handleDynamicWebhook).Methods(methods...)
		}
	}

	// Custom 404 handler
	r.NotFoundHandler = http.HandlerFunc(ws.notFoundHandler)

	return r
}

func (ws *WebhookServer) healthCheck(w http.ResponseWriter, _ *http.Request) {
	status := map[string]interface{}{
		"status":       "healthy",
		"destinations": len(ws.Config.Destinations),
		"routes":       len(ws.Config.Routes),
		"timestamp":    time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (ws *WebhookServer) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	ws.logger.Warn("Route not found",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`<html><head><title>404 Not Found</title></head><body><center><h1>404 Not Found</h1></center><hr><center>server</center></body></html>`))
}

func (ws *WebhookServer) validateRequest(r *http.Request) error {
	// Limit request body size (10MB)
	if r.ContentLength > 10<<20 {
		return fmt.Errorf("request body too large")
	}
	return nil
}

func (ws *WebhookServer) writeErrorResponse(w http.ResponseWriter, statusCode int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error: message,
		Code:  code,
	}
	json.NewEncoder(w).Encode(response)
}
