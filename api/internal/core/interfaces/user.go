package interfaces

import "github.com/google/uuid"

type User interface {
	GetID() uuid.UUID
	GetCNPJ() string
	GetIsAtivo() bool
	IsAnonymous() bool
}

type anonymousUser struct{}

func (anonymousUser) GetID() uuid.UUID  { return uuid.Nil }
func (anonymousUser) GetCNPJ() string   { return "" }
func (anonymousUser) GetIsAtivo() bool  { return false }
func (anonymousUser) IsAnonymous() bool { return true }

var AnonymousUser User = anonymousUser{}
