package rest

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type Responder interface {
	ErrorBadRequest(w http.ResponseWriter, err error)
	ErrorInternal(w http.ResponseWriter, err error)
	ErrorNotFound(w http.ResponseWriter)
	ErrorUnauthorized(w http.ResponseWriter, err error)
	ErrorConflict(w http.ResponseWriter, err error)
	OutputJSON(w http.ResponseWriter, responseData interface{})
	StatusCreated(w http.ResponseWriter, responseData interface{})
	StatusDeleted(w http.ResponseWriter)
}

type APIResponse struct {
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Respond struct {
	logger *zap.Logger
}

func NewResponder(logger *zap.Logger) Responder {
	return &Respond{logger: logger}
}

func (r *Respond) ErrorBadRequest(w http.ResponseWriter, err error) {
	APIResponse := APIResponse{
		Code:    http.StatusBadRequest,
		Type:    "Bad request",
		Message: err.Error(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(APIResponse.Code)

	err = json.NewEncoder(w).Encode(APIResponse)
	if err != nil {
		r.logger.Error("encode json response", zap.Error(err))
	}
}

// ErrorInternal logs the real error and returns a generic message:
// internal details (SQL, hosts, stack of wrapped calls) must not reach the client.
func (r *Respond) ErrorInternal(w http.ResponseWriter, err error) {
	r.logger.Error("internal error", zap.Error(err))

	APIResponse := APIResponse{
		Code:    http.StatusInternalServerError,
		Type:    "Internal error",
		Message: "internal server error",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(APIResponse.Code)

	err = json.NewEncoder(w).Encode(APIResponse)
	if err != nil {
		r.logger.Error("encode json response", zap.Error(err))
	}
}

func (r *Respond) ErrorNotFound(w http.ResponseWriter) {
	APIResponse := APIResponse{
		Code:    http.StatusNotFound,
		Type:    "Not found",
		Message: "Object not found",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(APIResponse.Code)

	err := json.NewEncoder(w).Encode(APIResponse)
	if err != nil {
		r.logger.Error("encode json response", zap.Error(err))
	}
}

func (r *Respond) ErrorUnauthorized(w http.ResponseWriter, err error) {
	APIResponse := APIResponse{
		Code:    http.StatusUnauthorized,
		Type:    "Authorization required",
		Message: err.Error(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(APIResponse.Code)

	err = json.NewEncoder(w).Encode(APIResponse)
	if err != nil {
		r.logger.Error("encode json response", zap.Error(err))
	}
}

func (r *Respond) ErrorConflict(w http.ResponseWriter, err error) {
	APIResponse := APIResponse{
		Code:    http.StatusConflict,
		Type:    "Conflict",
		Message: err.Error(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(APIResponse.Code)

	err = json.NewEncoder(w).Encode(APIResponse)
	if err != nil {
		r.logger.Error("encode json response", zap.Error(err))
	}
}

func (r *Respond) OutputJSON(w http.ResponseWriter, responseData interface{}) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(responseData)
	if err != nil {
		r.logger.Error("encode json response", zap.Error(err))
	}
}

func (r *Respond) StatusCreated(w http.ResponseWriter, responseData interface{}) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)

	err := json.NewEncoder(w).Encode(responseData)
	if err != nil {
		r.logger.Error("encode json response", zap.Error(err))
	}
}

func (r *Respond) StatusDeleted(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
