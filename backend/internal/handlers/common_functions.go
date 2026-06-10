package handlers

import (
	"encoding/json"
	"net/http"
)


func WriteResponse(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func ReadRequest(r *http.Request, data any) error {
	return json.NewDecoder(r.Body).Decode(data)
}
