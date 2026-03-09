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
	nomeFantasia string    `db:"nome_fantasia"`
	razaoSocial  string    `db:"razao_social"`
	isAtivo      bool      `db:"is_ativo"`
	cnpj         string    `db:"cnpj"`
	email        string    `db:"email"`
}

type EmpresaDTO struct {
	BaseModel
	ID           *uuid.UUID
	NomeFantasia *string `json:"nomeFantasia"`
	RazaoSocial  *string `json:"razaoSocial"`
	Cnpj         *string `json:"cnpj"`
	Email        *string `json:"email"`
}

func (m Empresa) toDTO() *EmpresaDTO {
	return &EmpresaDTO{
		ID:           &m.ID,
		NomeFantasia: &m.nomeFantasia,
		RazaoSocial:  &m.razaoSocial,
		Cnpj:         &m.cnpj,
		Email:        &m.email,
	}
}

func (d EmpresaDTO) toModel() *Empresa {
	var model Empresa

	if d.ID != nil {
		model.ID = *d.ID
	}

	if d.NomeFantasia != nil {
		model.nomeFantasia = *d.NomeFantasia
	}

	if d.RazaoSocial != nil {
		model.razaoSocial = *d.RazaoSocial
	}

	if d.Cnpj != nil {
		model.cnpj = *d.Cnpj
	}

	if d.Email != nil {
		model.email = *d.Email
	}

	return &model
}

func (e *Empresa) Validate(v *validator.Validator) {
	v.Check(e.nomeFantasia != "", "nome_fantasia", "must be provided")
	v.Check(len(e.nomeFantasia) <= 100, "nome_fantasia", "must not be more than 100 characters long")

	v.Check(e.razaoSocial != "", "razao_social", "must be provided")
	v.Check(len(e.razaoSocial) <= 100, "razao_social", "must not be more than 100 characters long")

	v.Check(e.cnpj != "", "cnpj", "must be provided")
	v.Check(ValidateCNPJ(e.cnpj), "cnpj", "invalid CNPJ format or verification digits")

	v.Check(e.email != "", "email", "must be provided")
	v.Check(validator.Matches(e.email, validator.EmailRX), "email", "must be a valid email address")

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
