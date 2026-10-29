package rest

import (
	"company-registry/internal/application/rest/dto"
	"company-registry/internal/domain/entity"
)

func companyToDTOCompany(company entity.Company) dto.Company {
	return dto.Company{
		ID:                company.ID.String(),
		Name:              company.Name,
		Description:       company.Description,
		Type:              company.Type,
		AmountOfEmployees: company.AmountOfEmployees,
		Registered:        company.Registered,
	}
}

func DTOCompanyToCompanyPatch(company dto.PatchCompanyRequest) entity.CompanyPatch {
	return entity.CompanyPatch{
		Name:              company.Name,
		Description:       company.Description,
		Type:              company.Type,
		AmountOfEmployees: company.AmountOfEmployees,
		Registered:        company.Registered,
	}
}
