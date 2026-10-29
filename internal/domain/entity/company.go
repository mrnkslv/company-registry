package entity

import "github.com/google/uuid"

type Company struct {
	ID                uuid.UUID `db:"id"`
	Name              string    `db:"name"`
	Description       *string   `db:"description"`
	Type              string    `db:"type"`
	AmountOfEmployees int32     `db:"amount_of_employees"`
	Registered        bool      `db:"registered"`
}

type CompanyPatch struct {
	Name              *string
	Description       *string
	Type              *string
	AmountOfEmployees *int32
	Registered        *bool
}

func (c *Company) Apply(p CompanyPatch) {
	if p.Name != nil {
		c.Name = *p.Name
	}
	if p.Description != nil {
		c.Description = p.Description
	}
	if p.Type != nil {
		c.Type = *p.Type
	}
	if p.AmountOfEmployees != nil {
		c.AmountOfEmployees = *p.AmountOfEmployees
	}
	if p.Registered != nil {
		c.Registered = *p.Registered
	}
}
