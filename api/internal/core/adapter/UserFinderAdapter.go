package adapter

import (
	"gestaoVet/internal/core/interfaces"
	"gestaoVet/internal/core/validator"
	"gestaoVet/internal/features/usuario"
)

type UserFinderAdapter struct {
	Service usuario.UsuarioService
}

func (a UserFinderAdapter) FindByEmail(
	email string,
	v *validator.Validator,
) (interfaces.User, error) {
	return a.Service.FindByEmail(email, v)
}
