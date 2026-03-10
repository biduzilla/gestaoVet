package models

import (
	"gestaoVet/utils/validator"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Empresa struct {
	BaseModel
	ID           uuid.UUID `db:"id"`
	NomeFantasia string    `db:"nome_fantasia"`
	RazaoSocial  string    `db:"razao_social"`
	IsAtivo      bool      `db:"is_ativo"`
	Cnpj         string    `db:"cnpj"`
	Email        string    `db:"email"`
}

type EmpresaDTO struct {
	ID           *uuid.UUID
	NomeFantasia *string `json:"nomeFantasia"`
	RazaoSocial  *string `json:"razaoSocial"`
	Cnpj         *string `json:"cnpj"`
	Email        *string `json:"email"`
}

func (m Empresa) toDTO() *EmpresaDTO {
	return &EmpresaDTO{
		ID:           &m.ID,
		NomeFantasia: &m.NomeFantasia,
		RazaoSocial:  &m.RazaoSocial,
		Cnpj:         &m.Cnpj,
		Email:        &m.Email,
	}
}

func (d EmpresaDTO) toModel() *Empresa {
	var model Empresa

	if d.ID != nil {
		model.ID = *d.ID
	}

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

	return &model
}

func (e *Empresa) Validate(v *validator.Validator) {
	v.Check(e.NomeFantasia != "", "nome_fantasia", "must be provided")
	v.Check(len(e.NomeFantasia) <= 100, "nome_fantasia", "must not be more than 100 characters long")

	v.Check(e.RazaoSocial != "", "razao_social", "must be provided")
	v.Check(len(e.RazaoSocial) <= 100, "razao_social", "must not be more than 100 characters long")

	v.Check(e.Cnpj != "", "cnpj", "must be provided")
	v.Check(ValidateCNPJ(e.Cnpj), "cnpj", "invalid CNPJ format or verification digits")

	v.Check(e.Email != "", "email", "must be provided")
	v.Check(validator.Matches(e.Email, validator.EmailRX), "email", "must be a valid email address")

}

func ValidateCNPJ(cnpj string) bool {
	cnpj = cleanCNPJ(cnpj)

	if len(cnpj) != 14 {
		return false
	}

	if allDigitsEqual(cnpj) {
		return false
	}

	if !validateCNPJDigits(cnpj) {
		return false
	}

	return true
}

func cleanCNPJ(cnpj string) string {
	cnpj = strings.ReplaceAll(cnpj, ".", "")
	cnpj = strings.ReplaceAll(cnpj, "-", "")
	cnpj = strings.ReplaceAll(cnpj, "/", "")
	cnpj = strings.ReplaceAll(cnpj, " ", "")
	return cnpj
}

func allDigitsEqual(cnpj string) bool {
	firstDigit := cnpj[0]
	for i := 1; i < len(cnpj); i++ {
		if cnpj[i] != firstDigit {
			return false
		}
	}
	return true
}

func validateCNPJDigits(cnpj string) bool {
	digits := cnpj[:12]
	firstDigit := calculateCNPJDigit(digits, true)
	if firstDigit != int(cnpj[12]-'0') {
		return false
	}

	digits = cnpj[:13]
	secondDigit := calculateCNPJDigit(digits, false)
	if secondDigit != int(cnpj[13]-'0') {
		return false
	}

	return true
}

func calculateCNPJDigit(base string, isFirst bool) int {
	var pesoInicial int
	if isFirst {
		pesoInicial = 5
	} else {
		pesoInicial = 6
	}

	soma := 0
	peso := pesoInicial

	for i := 0; i < len(base); i++ {
		num, _ := strconv.Atoi(string(base[i]))
		soma += num * peso
		peso--

		if peso < 2 {
			peso = 9
		}
	}

	resto := soma % 11
	if resto < 2 {
		return 0
	}
	return 11 - resto
}
