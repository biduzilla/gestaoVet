package models

import (
	"gestaoVet/internal/core/interfaces"

	"github.com/google/uuid"
)

type authenticatedUser struct {
	id       uuid.UUID
	username string
	cnpj     string
	isAtivo  bool
	roles    []interfaces.Role
}

func (u *authenticatedUser) GetID() uuid.UUID            { return u.id }
func (u *authenticatedUser) GetUsername() string         { return u.username }
func (u *authenticatedUser) GetCNPJ() string             { return u.cnpj }
func (u *authenticatedUser) GetIsAtivo() bool            { return u.isAtivo }
func (u *authenticatedUser) IsAnonymous() bool           { return false }
func (u *authenticatedUser) GetRoles() []interfaces.Role { return u.roles }

func NewAuthenticatedUser(
	id uuid.UUID,
	username string,
	cnpj string,
	isAtivo bool,
	roles []interfaces.Role,
) interfaces.User {
	return &authenticatedUser{
		id:       id,
		username: username,
		cnpj:     cnpj,
		isAtivo:  isAtivo,
		roles:    roles,
	}
}
