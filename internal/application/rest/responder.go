package rest

import (
	"encoding/json"
	"log"
	"net/http"
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

type ApiResponse struct {
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Respond struct{}

func NewResponder() Responder {
	return &Respond{}
}

func (r *Respond) ErrorBadRequest(w http.ResponseWriter, err error) {
	ApiResponse := ApiResponse{
		Code:    http.StatusBadRequest,
		Type:    "Bad request",
		Message: err.Error(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ApiResponse.Code)
	err = json.NewEncoder(w).Encode(ApiResponse)
	if err != nil {
		log.Println("encode json error: ", err)
	}
}

func (r *Respond) ErrorInternal(w http.ResponseWriter, err error) {
	ApiResponse := ApiResponse{
		Code:    http.StatusInternalServerError,
		Type:    "Internal error",
		Message: err.Error(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ApiResponse.Code)
	err = json.NewEncoder(w).Encode(ApiResponse)
	if err != nil {
		log.Println("encode json error: ", err)
	}
}

func (r *Respond) ErrorNotFound(w http.ResponseWriter) {
	ApiResponse := ApiResponse{
		Code:    http.StatusNotFound,
		Type:    "Not found",
		Message: "Object not found",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ApiResponse.Code)
	err := json.NewEncoder(w).Encode(ApiResponse)
	if err != nil {
		log.Println("encode json error: ", err)
	}
}

func (r *Respond) ErrorUnauthorized(w http.ResponseWriter, err error) {
	ApiResponse := ApiResponse{
		Code:    http.StatusUnauthorized,
		Type:    "Authorization required",
		Message: err.Error(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ApiResponse.Code)
	err = json.NewEncoder(w).Encode(ApiResponse)
	if err != nil {
		log.Println("encode json error: ", err)
	}
}

func (r *Respond) ErrorConflict(w http.ResponseWriter, err error) {
	ApiResponse := ApiResponse{
		Code:    http.StatusConflict,
		Type:    "Conflict",
		Message: err.Error(),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ApiResponse.Code)

	err = json.NewEncoder(w).Encode(ApiResponse)
	if err != nil {
		log.Println("encode json error: ", err)
	}
}

func (r *Respond) OutputJSON(w http.ResponseWriter, responseData interface{}) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(responseData)
	if err != nil {
		log.Println("encode json error: ", err)
	}
}

func (r *Respond) StatusCreated(w http.ResponseWriter, responseData interface{}) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)

	err := json.NewEncoder(w).Encode(responseData)
	if err != nil {
		log.Println("encode json error: ", err)
	}
}

func (r *Respond) StatusDeleted(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
