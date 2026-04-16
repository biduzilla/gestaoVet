package tutor

import (
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/domain/models"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"

	"github.com/google/uuid"
)

type Tutor struct {
	models.BaseModelCnpj
	ID          uuid.UUID
	Nome        string `db:"nome"`
	Celular     string `db:"celular"`
	Sexo        string `db:"sexo"`
	Nascimento  string `db:"nascimento"`
	Identidade  string `db:"identidade"`
	CPF         string `db:"cpf"`
	Observacoes string `db:"observacoes"`
	CEP         string `db:"cep"`
	Endereco    string `db:"endereco"`
	Bairro      string `db:"bairro"`
	Cidade      string `db:"cidade"`
	Telefone1   string `db:"telefone1"`
	Telefone2   string `db:"telefone2"`
	Email1      string `db:"email1"`
	Email2      string `db:"email2"`
	Numero      string `db:"numero"`
	Complemento string `db:"complemento"`
	Estado      string `db:"estado"`
}

type TutorDTO struct {
	ID          *uuid.UUID `json:"id"`
	Nome        *string    `json:"nome"`
	Celular     *string    `json:"celular"`
	Sexo        *string    `json:"sexo"`
	Nascimento  *string    `json:"nascimento"`
	Identidade  *string    `json:"identidade"`
	CPF         *string    `json:"cpf"`
	Observacoes *string    `json:"observacoes"`
	CEP         *string    `json:"cep"`
	Endereco    *string    `json:"endereco"`
	Bairro      *string    `json:"bairro"`
	Cidade      *string    `json:"cidade"`
	Telefone1   *string    `json:"telefone1"`
	Telefone2   *string    `json:"telefone2"`
	Email1      *string    `json:"email1"`
	Email2      *string    `json:"email2"`
	Numero      *string    `json:"numero"`
	Complemento *string    `json:"complemento"`
	Estado      *string    `json:"estado"`
	Cnpj        *string    `json:"cnpj"`
	Version     *int       `json:"version"`
}

func (t Tutor) ToDTO() *TutorDTO {
	return &TutorDTO{
		ID:          &t.ID,
		Nome:        &t.Nome,
		Celular:     &t.Celular,
		Sexo:        &t.Sexo,
		Nascimento:  &t.Nascimento,
		Identidade:  &t.Identidade,
		CPF:         &t.CPF,
		Observacoes: &t.Observacoes,
		CEP:         &t.CEP,
		Endereco:    &t.Endereco,
		Bairro:      &t.Bairro,
		Cidade:      &t.Cidade,
		Telefone1:   &t.Telefone1,
		Telefone2:   &t.Telefone2,
		Email1:      &t.Email1,
		Email2:      &t.Email2,
		Numero:      &t.Numero,
		Complemento: &t.Complemento,
		Estado:      &t.Estado,
		Cnpj:        &t.Cnpj,
		Version:     &t.Version,
	}
}

func (d TutorDTO) ToModel(v *validator.Validator) (*Tutor, error) {
	var model Tutor

	if d.ID != nil {
		model.ID = *d.ID
	}

	if d.Nome != nil {
		model.Nome = *d.Nome
	}

	if d.Celular != nil {
		model.Celular = *d.Celular
	}

	if d.Sexo != nil {
		model.Sexo = *d.Sexo
	}

	if d.Nascimento != nil {
		model.Nascimento = *d.Nascimento
	}

	if d.Identidade != nil {
		model.Identidade = *d.Identidade
	}

	if d.CPF != nil {
		v.Check(utils.ValidateCPF(*d.CPF), "cpf", "invalid cpf format")
		if !v.Valid() {
			return nil, e.ErrInvalidData
		}
		model.CPF = *d.CPF
	}

	if d.Observacoes != nil {
		model.Observacoes = *d.Observacoes
	}

	if d.CEP != nil {
		model.CEP = *d.CEP
	}

	if d.Endereco != nil {
		model.Endereco = *d.Endereco
	}

	if d.Bairro != nil {
		model.Bairro = *d.Bairro
	}

	if d.Cidade != nil {
		model.Cidade = *d.Cidade
	}

	if d.Telefone1 != nil {
		model.Telefone1 = *d.Telefone1
	}

	if d.Telefone2 != nil {
		model.Telefone2 = *d.Telefone2
	}

	if d.Email1 != nil {
		model.Email1 = *d.Email1
	}

	if d.Email2 != nil {
		model.Email2 = *d.Email2
	}

	if d.Numero != nil {
		model.Numero = *d.Numero
	}

	if d.Complemento != nil {
		model.Complemento = *d.Complemento
	}

	if d.Estado != nil {
		model.Estado = *d.Estado
	}

	if d.Cnpj != nil {
		v.Check(utils.ValidateCNPJ(*d.Cnpj), "cnpj", "invalid cnpj format")
		if !v.Valid() {
			return nil, e.ErrInvalidData
		}
		model.Cnpj = *d.Cnpj
	}

	if d.Version != nil {
		model.Version = *d.Version
	}

	return &model, nil
}

func (t *Tutor) Validate(v *validator.Validator) {
	v.Check(t.Nome != "", "nome", "must be provided")
	v.Check(len(t.Nome) >= 3, "nome", "must be at least 3 characters long")
	v.Check(len(t.Nome) <= 100, "nome", "must not be more than 100 characters long")

	v.Check(t.CPF != "", "cpf", "must be provided")
	v.Check(utils.ValidateCPF(t.CPF), "cpf", "invalid cpf format")

	v.Check(t.Celular != "", "celular", "must be provided")
	v.Check(utils.ValidateTelefone(t.Celular), "celular", "invalid phone format")

	if t.Sexo != "" {
		v.Check(t.Sexo == "M" || t.Sexo == "F" || t.Sexo == "Outro", "sexo", "must be 'M', 'F' or 'Outro'")
	}

	if t.Nascimento != "" {
		v.Check(utils.ValidateDate(t.Nascimento), "nascimento", "invalid date format (DD/MM/YYYY)")
	}

	if t.CEP != "" {
		v.Check(utils.ValidateCEP(t.CEP), "cep", "invalid cep format")
	}

	if t.Telefone1 != "" {
		v.Check(utils.ValidateTelefone(t.Telefone1), "telefone1", "invalid phone format")
	}

	if t.Telefone2 != "" {
		v.Check(utils.ValidateTelefone(t.Telefone2), "telefone2", "invalid phone format")
	}

	if t.Email1 != "" {
		v.Check(validator.Matches(t.Email1, validator.EmailRX), "email1", "must be a valid email address")
	}

	if t.Email2 != "" {
		v.Check(validator.Matches(t.Email2, validator.EmailRX), "email2", "must be a valid email address")
	}

	v.Check(t.Email1 != "" || t.Telefone1 != "", "contato", "at least one contact method must be provided")

	if t.Estado != "" {
		v.Check(len(t.Estado) == 2, "estado", "must be exactly 2 characters long")
	}
}

func ValidateCPF(v *validator.Validator, cpf string) {
	v.Check(cpf != "", "cpf", "must be provided")
	v.Check(utils.ValidateCPF(cpf), "cpf", "invalid cpf format")
}

func ValidateDataNascimento(v *validator.Validator, data string) {
	v.Check(data != "", "nascimento", "must be provided")
	v.Check(utils.ValidateDate(data), "nascimento", "invalid date format (DD/MM/YYYY)")
}
