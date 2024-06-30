package restfulApi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type JsonResponse[T any] struct {
	Data T `json:"data,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (j *JsonResponse[T]) SendResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(j)
	if err != nil {
		fmt.Println("Unable to encode response")
		fmt.Println(err)
		ReturnErrorResponse(w, "failed to encode response")
	}
}

func ReturnErrorResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusBadRequest)
	errorResponse := ErrorResponse{Error: message}
	err := json.NewEncoder(w).Encode(errorResponse)
	if err != nil {
		fmt.Println("Unable to encode error response")
	}
}
