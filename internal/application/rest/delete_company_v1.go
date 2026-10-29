package rest

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// DeleteCompanyV1 godoc
// @Summary      Delete company by ID
// @Tags         Company
// @Produce      json
// @Param        id   path      string  true  "Company ID"
// @Success      204  {object}  nil
// @Failure      400  {object}  error
// @Failure      500  {object}  error
// @Router       /company/v1/{id} [delete]
func (s *Server) DeleteCompanyV1(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.reply.ErrorBadRequest(w, fmt.Errorf("invalid company id: %w", err))
		return
	}

	err = s.companyService.DeleteCompany(ctx, companyID)
	if err != nil {
		s.replyError(w, err)

		return
	}

	s.reply.StatusDeleted(w)
}
