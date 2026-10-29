package rest

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/samber/lo"

	_ "company-registry/internal/application/rest/dto"
)

// GetCompanyV1 godoc
// @Summary      Get company by ID
// @Tags         Company
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Success      200  {object}  dto.Company
// @Failure      400  {object}  error
// @Failure      404  {object}  error
// @Failure      500  {object}  error
// @Router       /company/v1/{id} [get]
func (s *Server) GetCompanyV1(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.reply.ErrorBadRequest(w, fmt.Errorf("invalid company id: %w", err))
		return
	}

	company, err := s.companyService.GetCompany(ctx, companyID)
	if err != nil {
		s.replyError(w, err)

		return
	}

	s.reply.OutputJSON(w, companyToDTOCompany(lo.FromPtr(company)))
}
