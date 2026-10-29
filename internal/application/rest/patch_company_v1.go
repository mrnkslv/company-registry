package rest

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"company-registry/internal/application/rest/dto"
	"company-registry/internal/domain/entity"
)

// PatchCompanyV1 godoc
// @Summary      Patch company by ID
// @Tags         Company
// @Produce      json
// @Accept       json
// @Success      200      {object}  dto.Company
// @Failure      400  {object}  error
// @Failure      404  {object}  error
// @Failure      409  {object}  error
// @Failure      500  {object}  error
// @Param        id       path  string                    true  "Company ID"
// @Param        company  body  dto.PatchCompanyRequest   true  "Fields to update"
// @Router       /company/v1/{id} [patch].
func (s *Server) PatchCompanyV1(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.reply.ErrorBadRequest(w, fmt.Errorf("invalid company id: %w", err))

		return
	}

	var request dto.PatchCompanyRequest
	if err := s.readRequest(r, &request); err != nil {
		s.reply.ErrorBadRequest(w, fmt.Errorf("readRequest: %w", err))

		return
	}

	company, err := s.companyService.PatchCompany(ctx, id, entity.CompanyPatch{
		Name:              request.Name,
		Description:       request.Description,
		Type:              request.Type,
		AmountOfEmployees: request.AmountOfEmployees,
		Registered:        request.Registered,
	})
	if err != nil {
		s.replyError(w, err)

		return
	}

	s.reply.OutputJSON(w, companyToDTOCompany(*company))
}
