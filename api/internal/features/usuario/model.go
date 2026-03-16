package usuario

import (
	"gestaoVet/internal/core/domain/models"
	"gestaoVet/internal/core/validator"
	"regexp"

	"github.com/google/uuid"
)

type Usuario struct {
	models.BaseModel
	ID       uuid.UUID `db:"id"`
	Nome     string    `db:"nome"`
	Telefone string    `db:"telefone"`
	Email    string    `db:"email"`
	Senha    string    `db:"email"`
	IsAtivo  bool      `db:"is_ativo"`
}

type UsuarioDTO struct {
	ID       *uuid.UUID `json:"id"`
	Nome     *string    `json:"nome"`
	Telefone *string    `json:"telefone"`
	Email    *string    `json:"email"`
	Cnpj     *string    `json:"cnpj"`
	Senha    *string    `json:"-"`
}

func (m Usuario) ToDTO() *UsuarioDTO {
	return &UsuarioDTO{
		ID:       &m.ID,
		Nome:     &m.Nome,
		Telefone: &m.Telefone,
		Email:    &m.Email,
	}
}

func (d UsuarioDTO) ToModel() *Usuario {
	var model Usuario

	if d.ID != nil {
		model.ID = *d.ID
	}

	if d.Nome != nil {
		model.Nome = *d.Nome
	}

	if d.Telefone != nil {
		model.Telefone = *d.Telefone
	}

	if d.Email != nil {
		model.Email = *d.Email
	}

	if d.Senha != nil {
		model.Senha = *d.Senha
	}

	return &model
}

func (u *Usuario) Validate(v *validator.Validator) {
	v.Check(u.Nome != "", "nome", "must be provided")
	v.Check(len(u.Nome) >= 3, "nome", "must be at least 3 characters long")
	v.Check(len(u.Nome) <= 100, "nome", "must not be more than 100 characters long")
	v.Check(u.Telefone != "", "telefone", "must be provided")
	v.Check(ValidateTelefone(u.Telefone), "telefone", "invalid telephone format")
	v.Check(u.Email != "", "email", "must be provided")
	v.Check(validator.Matches(u.Email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidateTelefone(telefone string) bool {
	telefone = regexp.MustCompile(`[^\d]`).ReplaceAllString(telefone, "")

	switch len(telefone) {
	case 8, 9, 10, 11:
		return true
	default:
		return false
	}
}

func (u *Usuario) SetSenha() {
}

func (u *Usuario) CheckSenha() bool {
	return true
}
