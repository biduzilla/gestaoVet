package permissao

import (
	"gestaoVet/internal/core/domain/models"
	"gestaoVet/internal/core/validator"

	"github.com/google/uuid"
)

type Recurso string
type Acao string

const (
	usuarioRecurso Recurso = "usuario_recurso"
	empresaRecurso Recurso = "empresa_recurso"
)

const (
	createAcao Acao = "create_acao"
	readAcao   Acao = "read_acao"
	updateAcao Acao = "update_acao"
	deleteAcao Acao = "delete_acao"
)

type Permissao struct {
	ID      uuid.UUID `db:"id"`
	Recurso Recurso   `db:"recurso"`
	Acao    Acao      `db:"acao"`
}

type Cargo struct {
	models.BaseModelCnpj
	ID         uuid.UUID   `db:"id"`
	Nome       string      `db:"nome"`
	Permissoes []Permissao `db:"Permissoes"`
}

type PermissaoDTO struct {
	ID      *uuid.UUID `json:"id"`
	Recurso *Recurso   `json:"recurso"`
	Acao    *Acao      `json:"acao"`
}

type CargoDTO struct {
	ID         *uuid.UUID      `json:"id"`
	Nome       *string         `json:"nome"`
	Permissoes []*PermissaoDTO `json:"Permissoes"`
}

func (m Permissao) toDTO() *PermissaoDTO {
	return &PermissaoDTO{
		ID:      &m.ID,
		Recurso: &m.Recurso,
		Acao:    &m.Acao,
	}
}

func (d PermissaoDTO) toModel() *Permissao {
	var model Permissao
	if d.ID != nil {
		model.ID = *d.ID
	}

	if d.Recurso != nil {
		model.Recurso = *d.Recurso
	}

	if d.Acao != nil {
		model.Acao = *d.Acao
	}

	return &model
}

func (m Cargo) toDTO() *CargoDTO {
	var list = make([]*PermissaoDTO, 0, len(m.Permissoes))
	for i := range m.Permissoes {
		list = append(list, m.Permissoes[i].toDTO())
	}

	return &CargoDTO{
		ID:         &m.ID,
		Nome:       &m.Nome,
		Permissoes: list,
	}
}

func (d CargoDTO) toModel() *Cargo {
	var model Cargo

	if d.ID != nil {
		model.ID = *d.ID
	}

	if d.Nome != nil {
		model.Nome = *d.Nome
	}

	if d.Permissoes != nil {
		var list = make([]Permissao, len(d.Permissoes))
		for i := range d.Permissoes {
			list = append(list, *d.Permissoes[i].toModel())
		}
		model.Permissoes = list
	}

	return &model
}

func (p *Permissao) Validate(v *validator.Validator) {
	v.Check(p.Recurso != "", "recurso", "must be provided")
	v.Check(isValidRecurso(p.Recurso), "recurso", "invalid recurso")

	v.Check(p.Acao != "", "acao", "must be provided")
	v.Check(isValidAcao(p.Acao), "acao", "invalid acao")
}

func (c *Cargo) Validate(v *validator.Validator) {
	v.Check(c.Nome != "", "nome", "must be provided")
	v.Check(len(c.Nome) >= 3, "nome", "must be at least 3 characters long")
	v.Check(len(c.Nome) <= 50, "nome", "must not be more than 50 characters long")

	v.Check(c.Permissoes != nil, "permissoes", "must be provided")

	for i := range c.Permissoes {
		c.Permissoes[i].Validate(v)
	}
}

func isValidRecurso(r Recurso) bool {
	switch r {
	case usuarioRecurso, empresaRecurso:
		return true
	default:
		return false
	}
}

func isValidAcao(a Acao) bool {
	switch a {
	case createAcao, readAcao, updateAcao, deleteAcao:
		return true
	default:
		return false
	}
}
