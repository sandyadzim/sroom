package model

type Role string

const (
	RoleCustomer Role = "CUSTOMER"
	RoleAdmin    Role = "ADMIN"
	RoleOwner    Role = "OWNER"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleCustomer, RoleAdmin, RoleOwner:
		return true
	default:
		return false
	}
}
