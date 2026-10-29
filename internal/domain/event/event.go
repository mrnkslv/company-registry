package event

import (
	"github.com/google/uuid"

	"company-registry/internal/domain/entity"
)

type Type string

const (
	CompanyCreated Type = "company.created"
	CompanyUpdated Type = "company.updated"
	CompanyDeleted Type = "company.deleted"
)

// Company is the company snapshot published in every event.
// It is a separate type so the public event format does not change when the entity does.
type Company struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Description       *string   `json:"description,omitempty"`
	Type              string    `json:"type"`
	AmountOfEmployees int32     `json:"amount_of_employees"`
	Registered        bool      `json:"registered"`
}

func CompanyFromEntity(c *entity.Company) Company {
	return Company{
		ID:                c.ID,
		Name:              c.Name,
		Description:       c.Description,
		Type:              c.Type,
		AmountOfEmployees: c.AmountOfEmployees,
		Registered:        c.Registered,
	}
}
