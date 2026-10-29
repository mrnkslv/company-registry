package dto

type Company struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Description       *string `json:"description,omitempty"`
	Type              string  `json:"type"`
	AmountOfEmployees int32   `json:"amount_of_employees"`
	Registered        bool    `json:"registered"`
}

type CreateCompanyRequest struct {
	Name              string  `json:"name"`
	Description       *string `json:"description,omitempty"`
	Type              string  `json:"type"`
	AmountOfEmployees int32   `json:"amount_of_employees"`
	Registered        bool    `json:"registered"`
}

type PatchCompanyRequest struct {
	Name              *string `json:"name,omitempty"`
	Description       *string `json:"description,omitempty"`
	Type              *string `json:"type,omitempty"`
	AmountOfEmployees *int32  `json:"amount_of_employees,omitempty"`
	Registered        *bool   `json:"registered,omitempty"`
}
