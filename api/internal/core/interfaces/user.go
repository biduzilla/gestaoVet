package interfaces

import "github.com/google/uuid"

type Role int

const (
	ROLE_ADMIN Role = iota + 1
	ROLE_VET
	ROLE_ASSISTANT
	ROLE_RECEPTIONIST
)

type User interface {
	GetID() uuid.UUID
	GetCNPJ() string
	GetIsAtivo() bool
	IsAnonymous() bool
	GetRoles() []Role
}

type anonymousUser struct{}

func (anonymousUser) GetID() uuid.UUID  { return uuid.Nil }
func (anonymousUser) GetCNPJ() string   { return "" }
func (anonymousUser) GetIsAtivo() bool  { return false }
func (anonymousUser) IsAnonymous() bool { return true }
func (anonymousUser) GetRoles() []Role  { return []Role{} }

var AnonymousUser User = anonymousUser{}
