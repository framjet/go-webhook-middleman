package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/framjet/go-webhook-middleman/internal/config"
	"net/http"
	"text/template"
	"time"
)

// ResponseData holds all data available for templating
type ResponseData struct {
	Destinations []config.ResolvedDestination
	SuccessCount int
	Duration     time.Duration
	Results      []config.ForwardResult

	Params    map[string]string
	Variables map[string]string
	Body      string

	Request *http.Request `json:"-"` // Original request for context

	// Computed fields for convenience in templates
	ForwardedTo int   `json:"forwarded_to"`
	DurationMs  int64 `json:"duration_ms"`
	Successful  int   `json:"successful"`
}

// ResponseHandler handles response templating and sending
type ResponseHandler struct {
	route *config.Route
	data  *ResponseData
}

// NewResponseHandler creates a new response handler
func NewResponseHandler(route *config.Route, data *ResponseData) *ResponseHandler {
	// Add computed fields
	data.ForwardedTo = len(data.Destinations)
	data.DurationMs = data.Duration.Milliseconds()
	data.Successful = data.SuccessCount

	return &ResponseHandler{
		route: route,
		data:  data,
	}
}

// SendResponse sends the response with optional templating
func (rh *ResponseHandler) SendResponse(w http.ResponseWriter) error {
	// Determine status code
	statusCode := rh.getStatusCode()

	// Set headers
	if err := rh.setHeaders(w); err != nil {
		return fmt.Errorf("failed to set headers: %w", err)
	}

	// Set status code
	w.WriteHeader(statusCode)

	// Write body
	return rh.writeBody(w)
}

// getStatusCode determines the appropriate status code
func (rh *ResponseHandler) getStatusCode() int {
	if rh.data.SuccessCount == rh.data.ForwardedTo {
		if rh.route.Response != nil && rh.route.Response.Status != nil && rh.route.Response.Status.Success != nil {
			return *rh.route.Response.Status.Success
		}

		return http.StatusOK
	}

	if rh.route.Response != nil && rh.route.Response.Status != nil && rh.route.Response.Status.Failure != nil {
		return *rh.route.Response.Status.Failure
	}

	return http.StatusBadGateway
}

// setHeaders sets response headers with optional templating
func (rh *ResponseHandler) setHeaders(w http.ResponseWriter) error {
	// Set default content type if no custom response is configured
	if rh.route.Response == nil || rh.route.Response.Headers == nil {
		w.Header().Set("Content-Type", "application/json")
		return nil
	}

	// Apply templated headers
	for key, valueTemplate := range *rh.route.Response.Headers {
		value, err := rh.executeTemplate(valueTemplate)
		if err != nil {
			return fmt.Errorf("failed to template header %s: %w", key, err)
		}
		w.Header().Set(key, value)
	}

	// Set default content type if not specified
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}

	return nil
}

// writeBody writes the response body with optional templating
func (rh *ResponseHandler) writeBody(w http.ResponseWriter) error {
	// If no custom body template, use default JSON response
	if rh.route.Response == nil || rh.route.Response.Body == nil {
		return rh.writeDefaultJSONResponse(w)
	}

	// Use templated body
	body, err := rh.executeTemplate(*rh.route.Response.Body)
	if err != nil {
		return fmt.Errorf("failed to template body: %w", err)
	}

	_, err = w.Write([]byte(body))
	return err
}

// writeDefaultJSONResponse writes the default JSON response
func (rh *ResponseHandler) writeDefaultJSONResponse(w http.ResponseWriter) error {
	response := map[string]interface{}{
		"forwarded_to": rh.data.ForwardedTo,
		"successful":   rh.data.Successful,
		"duration_ms":  rh.data.DurationMs,
		"results":      rh.data.Results,
	}

	return json.NewEncoder(w).Encode(response)
}

// executeTemplate executes a template string with the response data
func (rh *ResponseHandler) executeTemplate(templateStr string) (string, error) {
	tmpl, err := template.New("response").Funcs(sprig.FuncMap()).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	ctx := map[string]interface{}{
		"params":       rh.data.Params,
		"var":          rh.data.Variables,
		"body":         rh.data.Body,
		"destinations": rh.data.Destinations,
		"successCount": rh.data.SuccessCount,
		"duration":     rh.data.Duration,
		"results":      rh.data.Results,
		"request":      rh.data.Request, // Include original request for context
		"forwardedTo":  rh.data.ForwardedTo,
		"durationMs":   rh.data.DurationMs,
		"successful":   rh.data.Successful,
	}
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
