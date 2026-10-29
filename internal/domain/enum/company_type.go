package enum

import "database/sql/driver"

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
