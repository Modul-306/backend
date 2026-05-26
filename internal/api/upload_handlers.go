package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "File too large")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid file")
		return
	}
	defer file.Close()

	url, err := s.storage.UploadFile(r.Context(), "cattlehof", header.Filename, file)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to upload file")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}
