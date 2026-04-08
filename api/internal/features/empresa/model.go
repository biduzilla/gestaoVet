package empresa

import (
	"gestaoVet/internal/core/domain/models"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"
)

type Empresa struct {
	models.BaseModel
	Cnpj         string `db:"cnpj"`
	NomeFantasia string `db:"nome_fantasia"`
	RazaoSocial  string `db:"razao_social"`
	Telefone     string `db:"telefone"`
	IsAtivo      bool   `db:"is_ativo"`
	Email        string `db:"email"`
}

type EmpresaDTO struct {
	NomeFantasia *string `json:"nomeFantasia"`
	RazaoSocial  *string `json:"razaoSocial"`
	Telefone     *string `json:"telefone"`
	Cnpj         *string `json:"cnpj"`
	Email        *string `json:"email"`
	Version      *int    `json:"version"`
}

func (m Empresa) toDTO() *EmpresaDTO {
	return &EmpresaDTO{
		Cnpj:         &m.Cnpj,
		NomeFantasia: &m.NomeFantasia,
		RazaoSocial:  &m.RazaoSocial,
		Email:        &m.Email,
		Telefone:     &m.Telefone,
		Version:      &m.Version,
	}
}

func (d EmpresaDTO) toModel() *Empresa {
	var model Empresa

	if d.NomeFantasia != nil {
		model.NomeFantasia = *d.NomeFantasia
	}

	if d.RazaoSocial != nil {
		model.RazaoSocial = *d.RazaoSocial
	}

	if d.Cnpj != nil {
		model.Cnpj = *d.Cnpj
	}

	if d.Email != nil {
		model.Email = *d.Email
	}

	if d.Telefone != nil {
		model.Telefone = *d.Telefone
	}

	if d.Version != nil {
		model.Version = *d.Version
	}

	return &model
}

func (e *Empresa) Validate(v *validator.Validator) {
	v.Check(e.NomeFantasia != "", "nome_fantasia", "must be provided")
	v.Check(len(e.NomeFantasia) <= 100, "nome_fantasia", "must not be more than 100 characters long")

	v.Check(e.RazaoSocial != "", "razao_social", "must be provided")
	v.Check(len(e.RazaoSocial) <= 100, "razao_social", "must not be more than 100 characters long")

	v.Check(e.Telefone != "", "telefone", "must be provided")
	v.Check(utils.ValidateTelefone(e.Telefone), "telefone", "invalid telephone format")

	v.Check(e.Cnpj != "", "cnpj", "must be provided")
	v.Check(utils.ValidateCNPJ(e.Cnpj), "cnpj", "invalid CNPJ format or verification digits")

	v.Check(e.Email != "", "email", "must be provided")
	v.Check(validator.Matches(e.Email, validator.EmailRX), "email", "must be a valid email address")

}
