package rest

import (
	"fmt"
	"net/http"

	"company-registry/internal/application/rest/dto"
)

// LoginV1 godoc
// @Summary      Get access token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      dto.LoginRequest  true  "Credentials"
// @Success      200          {object}  dto.LoginResponse
// @Failure      400          {object}  ApiResponse
// @Failure      401          {object}  ApiResponse
// @Failure      500          {object}  ApiResponse
// @Router       /auth/v1/login [post]
func (s *Server) LoginV1(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var request dto.LoginRequest
	if err := s.readRequest(r, &request); err != nil {
		s.reply.ErrorBadRequest(w, fmt.Errorf("readRequest: %w", err))
		return
	}

	token, err := s.authService.GenerateToken(ctx, request.Username, request.Password)
	if err != nil {
		s.replyError(w, err)
		return
	}

	s.reply.OutputJSON(w, dto.LoginResponse{Token: token})
}
