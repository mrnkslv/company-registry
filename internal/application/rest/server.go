package rest

import (
	_ "company-registry/docs"
	"company-registry/internal/config"
	"company-registry/internal/domain/entity"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"company-registry/internal/domain/errs"

	"github.com/google/uuid"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/go-chi/chi/v5"
)

type CompanyService interface {
	CreateCompany(ctx context.Context, company *entity.Company) (uuid.UUID, error)
	GetCompany(ctx context.Context, id uuid.UUID) (*entity.Company, error)
	PatchCompany(ctx context.Context, id uuid.UUID, patch entity.CompanyPatch) (*entity.Company, error)
	DeleteCompany(ctx context.Context, id uuid.UUID) error
}

type AuthService interface {
	GenerateToken(ctx context.Context, username, password string) (string, error)
}

type authMiddleware interface {
	Authenticate(next http.Handler) http.Handler
}

type Server struct {
	companyService CompanyService
	authService    AuthService
	authMiddleware authMiddleware
	reply          reply
	cfg            config.Config
}

func NewServer(
	companyService CompanyService,
	authService AuthService,
	authMiddleware authMiddleware,
	reply reply,
	cfg config.Config,
) *Server {
	return &Server{
		companyService: companyService,
		authService:    authService,
		authMiddleware: authMiddleware,
		reply:          reply,
		cfg:            cfg,
	}
}

type reply interface {
	ErrorBadRequest(w http.ResponseWriter, err error)
	ErrorInternal(w http.ResponseWriter, err error)
	ErrorNotFound(w http.ResponseWriter)
	ErrorUnauthorized(w http.ResponseWriter, err error)
	ErrorConflict(w http.ResponseWriter, err error)
	OutputJSON(w http.ResponseWriter, responseData interface{})
	StatusCreated(w http.ResponseWriter, responseData interface{})
	StatusDeleted(w http.ResponseWriter)
}

func (s Server) RegisterRoutes(
	r chi.Router,
) {
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"), // или относительный /swagger/doc.json
	))

	r.Post("/auth/v1/login", s.LoginV1)

	r.Route("/company/v1", func(r chi.Router) {
		r.Get("/{id}", s.GetCompanyV1)

		r.Group(func(r chi.Router) {
			r.Use(s.authMiddleware.Authenticate)

			r.Post("/", s.CreateCompanyV1)
			r.Patch("/{id}", s.PatchCompanyV1)
			r.Delete("/{id}", s.DeleteCompanyV1)
		})
	})
}

func (s Server) readRequest(r *http.Request, dest any) error {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		return fmt.Errorf("json.Decode: %w", err)
	}

	return nil
}

func (s *Server) replyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errs.ErrValidation):
		s.reply.ErrorBadRequest(w, err)
	case errors.Is(err, errs.ErrNotFound):
		s.reply.ErrorNotFound(w)
	case errors.Is(err, errs.ErrAlreadyExists):
		s.reply.ErrorConflict(w, err)
	case errors.Is(err, errs.ErrInvalidCredentials):
		s.reply.ErrorUnauthorized(w, err)
	default:
		s.reply.ErrorInternal(w, err)
	}
}
