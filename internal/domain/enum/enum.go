package enum

import (
	"database/sql/driver"
	"fmt"
)

// Enum is a base type for enum values.
type enum struct {
	value string
}

// String implements the fmt.Stringer interface.
func (e enum) String() string {
	return e.value
}

// Value implements the driver.Valuer interface.
func (e enum) Value() (driver.Value, error) {
	return e.value, nil
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
