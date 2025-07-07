package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/richard-senior/mcp/_digital-io/internal/config"
	"github.com/richard-senior/mcp/_digital-io/internal/iobank"
)

// APIHandler handles HTTP requests for the I/O bank
type APIHandler struct {
	ioBank *iobank.IOBank
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(bank *iobank.IOBank) *APIHandler {
	return &APIHandler{
		ioBank: bank,
	}
}

// SetupRoutes configures the HTTP routes
func (h *APIHandler) SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// REST endpoints
	r.HandleFunc("/status", h.handleStatus).Methods("GET")
	r.HandleFunc("/reset", h.handleReset).Methods("POST")
	r.HandleFunc("/digital/input/{pin}", h.handleGetDigitalInput).Methods("GET")
	r.HandleFunc("/digital/output/{pin}", h.handleSetDigitalOutput).Methods("POST")
	r.HandleFunc("/digital/output/{pin}", h.handleGetDigitalOutput).Methods("GET")
	r.HandleFunc("/analog/input/{pin}", h.handleGetAnalogInput).Methods("GET")
	r.HandleFunc("/analog/output/{pin}", h.handleSetAnalogOutput).Methods("POST")
	r.HandleFunc("/analog/output/{pin}", h.handleGetAnalogOutput).Methods("GET")

	// Label management endpoints
	r.HandleFunc("/labels", h.GetLabelsHandler).Methods("GET")
	r.HandleFunc("/labels/{type}/{pin}", h.UpdateLabelHandler).Methods("POST")
	r.HandleFunc("/labels/reload", h.ReloadLabelsHandler).Methods("POST")

	// MCP message recording endpoint
	r.HandleFunc("/mcp/message", h.handleRecordMCPMessage).Methods("POST")

	// Serve static files for a simple web interface
	webPath, err := config.GetWebPath()
	if err != nil {
		// Fallback to current working directory approach
		webPath = "web"
	}
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(webPath)))

	return r
}

// REST endpoint handlers
func (h *APIHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := h.ioBank.GetStatus()

	// Add labels to the status
	status = AddLabelsToStatus(status)

	json.NewEncoder(w).Encode(status)
}

func (h *APIHandler) handleGetDigitalInput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pin, err := strconv.Atoi(vars["pin"])
	if err != nil {
		http.Error(w, "Invalid pin number", http.StatusBadRequest)
		return
	}

	value, err := h.ioBank.GetDigitalInput(pin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pin":   pin,
		"value": value,
	})
}

func (h *APIHandler) handleSetDigitalOutput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pin, err := strconv.Atoi(vars["pin"])
	if err != nil {
		http.Error(w, "Invalid pin number", http.StatusBadRequest)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	value, ok := req["value"].(bool)
	if !ok {
		http.Error(w, "Missing or invalid 'value' field", http.StatusBadRequest)
		return
	}

	err = h.ioBank.SetDigitalOutput(pin, value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pin":    pin,
		"value":  value,
		"status": "success",
	})
}

func (h *APIHandler) handleGetDigitalOutput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pin, err := strconv.Atoi(vars["pin"])
	if err != nil {
		http.Error(w, "Invalid pin number", http.StatusBadRequest)
		return
	}

	value, err := h.ioBank.GetDigitalOutput(pin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pin":   pin,
		"value": value,
	})
}

func (h *APIHandler) handleGetAnalogInput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pin, err := strconv.Atoi(vars["pin"])
	if err != nil {
		http.Error(w, "Invalid pin number", http.StatusBadRequest)
		return
	}

	value, err := h.ioBank.GetAnalogInput(pin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pin":   pin,
		"value": fmt.Sprintf("%.3f", value),
		"unit":  "V",
	})
}

func (h *APIHandler) handleSetAnalogOutput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pin, err := strconv.Atoi(vars["pin"])
	if err != nil {
		http.Error(w, "Invalid pin number", http.StatusBadRequest)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	value, ok := req["value"].(float64)
	if !ok {
		http.Error(w, "Missing or invalid 'value' field", http.StatusBadRequest)
		return
	}

	err = h.ioBank.SetAnalogOutput(pin, value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pin":    pin,
		"value":  fmt.Sprintf("%.3f", value),
		"unit":   "V",
		"status": "success",
	})
}

func (h *APIHandler) handleGetAnalogOutput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pin, err := strconv.Atoi(vars["pin"])
	if err != nil {
		http.Error(w, "Invalid pin number", http.StatusBadRequest)
		return
	}

	value, err := h.ioBank.GetAnalogOutput(pin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"pin":   pin,
		"value": fmt.Sprintf("%.3f", value),
		"unit":  "V",
	})
}

func (h *APIHandler) handleReset(w http.ResponseWriter, r *http.Request) {
	// Reset the I/O bank to initial values
	err := h.ioBank.Reset()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset system: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "System reset to initial values",
	})
}

func (h *APIHandler) handleRecordMCPMessage(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	toolName, ok := req["tool_name"]
	if !ok {
		http.Error(w, "Missing 'tool_name' field", http.StatusBadRequest)
		return
	}

	message, ok := req["message"]
	if !ok {
		http.Error(w, "Missing 'message' field", http.StatusBadRequest)
		return
	}

	// Record the MCP message
	h.ioBank.AddMCPMessage(toolName, message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
	})
}
