package rest

import (
	"fmt"
	"net/http"

	"company-registry/internal/application/rest/dto"
	"company-registry/internal/domain/entity"
)

// CreateCompanyV1 godoc
// @Summary      Create new company
// @Tags         Company
// @Produce      json
// @Accept       json
// @Param        company   body      dto.CreateCompanyRequest  true  "Company"
// @Success      201  {object}  dto.Company
// @Failure      400  {object}  error
// @Failure      500  {object}  error
// @Router       /company/v1 [post].
func (s *Server) CreateCompanyV1(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var request dto.CreateCompanyRequest
	if err := s.readRequest(r, &request); err != nil {
		s.reply.ErrorBadRequest(w, fmt.Errorf("readRequest: %w", err))

		return
	}

	company := entity.Company{
		Name:              request.Name,
		Description:       request.Description,
		Type:              request.Type,
		AmountOfEmployees: request.AmountOfEmployees,
		Registered:        request.Registered,
	}

	companyID, err := s.companyService.CreateCompany(ctx, &company)
	if err != nil {
		s.replyError(w, err)

		return
	}

	s.reply.StatusCreated(w, dto.Company{
		ID:                companyID.String(),
		Name:              company.Name,
		Description:       company.Description,
		Type:              company.Type,
		AmountOfEmployees: company.AmountOfEmployees,
		Registered:        company.Registered,
	})
}
