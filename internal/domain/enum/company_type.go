package enum

import (
	"database/sql/driver"
	"fmt"
)

type CompanyType string

const (
	CompanyTypeCorporations       CompanyType = "Corporations"
	CompanyTypeNonProfit          CompanyType = "NonProfit"
	CompanyTypeCooperative        CompanyType = "Cooperative"
	CompanyTypeSoleProprietorship CompanyType = "Sole Proprietorship"
)

func (c CompanyType) String() string {
	return string(c)
}

func (c CompanyType) Value() (driver.Value, error) {
	return c.String(), nil
}

//nolint:cyclop
func ParseCompanyType(a any) (CompanyType, error) {
	s := fmt.Sprintf("%v", a)

	switch s {
	case CompanyTypeCorporations.String():
		return CompanyTypeCorporations, nil
	case CompanyTypeNonProfit.String():
		return CompanyTypeNonProfit, nil
	case CompanyTypeCooperative.String():
		return CompanyTypeCooperative, nil
	case CompanyTypeSoleProprietorship.String():
		return CompanyTypeSoleProprietorship, nil
	default:
		return "", fmt.Errorf("unsupported company type: %q", s)
	}
}
