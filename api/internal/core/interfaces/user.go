package interfaces

import "github.com/google/uuid"

type User interface {
	GetID() uuid.UUID
	GetCNPJ() string
	GetIsAtivo() bool
	IsAnonymous() bool
	GetRoles() []string
}

type anonymousUser struct{}

func (anonymousUser) GetID() uuid.UUID   { return uuid.Nil }
func (anonymousUser) GetCNPJ() string    { return "" }
func (anonymousUser) GetIsAtivo() bool   { return false }
func (anonymousUser) IsAnonymous() bool  { return true }
func (anonymousUser) GetRoles() []string { return []string{} }

var AnonymousUser User = anonymousUser{}
