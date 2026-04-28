package utils

import (
	"encoding/json"
	"gestaoVet/internal/core/interfaces"
	"maps"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Envelope map[string]any

func MinifySQL(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func WriteJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')
	maps.Copy(w.Header(), headers)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}

func GetTypeName(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return strings.ToLower(t.Name())
}

/*
func RunInTx(
	db any,
	fn func(tx *sql.Tx) error,
) error {
	if tx, ok := db.(*sql.Tx); ok {
		savepointName := fmt.Sprintf("sp_%d", time.Now().UnixNano())

		_, err := tx.Exec(fmt.Sprintf("SAVEPOINT %s", savepointName))
		if err != nil {
			return fmt.Errorf("failed to create savepoint: %w", err)
		}

		fnErr := fn(tx)
		if fnErr == nil {
			_, err = tx.Exec(fmt.Sprintf("RELEASE SAVEPOINT %s", savepointName))
			if err != nil {
				return fmt.Errorf("failed to release savepoint: %w", err)
			}
			return nil
		}

		_, rbErr := tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepointName))
		if rbErr != nil {
			return errors.Join(fnErr, fmt.Errorf("failed to rollback to savepoint: %w", rbErr))
		}

		return fnErr
	}

	if sqlDB, ok := db.(*sql.DB); ok {
		tx, err := sqlDB.Begin()
		if err != nil {
			return err
		}

		fnErr := fn(tx)
		if fnErr == nil {
			return tx.Commit()
		}

		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Join(fnErr, rbErr)
		}

		return fnErr
	}

	return fmt.Errorf("invalid db type: expected *sql.DB or *sql.Tx")
}
*/

func ConvertInt32ToRoles(rolesInt32 []int32) []interfaces.Role {
	roles := make([]interfaces.Role, len(rolesInt32))
	for i, r := range rolesInt32 {
		roles[i] = interfaces.Role(r)
	}
	return roles
}

func ConvertRolesToInt32(roles []interfaces.Role) []int32 {
	roles32 := make([]int32, len(roles))
	for i, r := range roles32 {
		roles[i] = interfaces.Role(r)
	}
	return roles32
}

func ValidateTelefone(telefone string) bool {
	telefone = regexp.MustCompile(`[^\d]`).ReplaceAllString(telefone, "")

	match, _ := regexp.MatchString(`^\d{2}9\d{8}$`, telefone)
	return match
}

func ValidateCPF(cpf string) bool {
	cpf = regexp.MustCompile(`[^0-9]`).ReplaceAllString(cpf, "")

	if len(cpf) != 11 {
		return false
	}

	todosIguais := true
	for i := 1; i < 11; i++ {
		if cpf[i] != cpf[0] {
			todosIguais = false
			break
		}
	}
	if todosIguais {
		return false
	}

	digitos := make([]int, 11)
	for i := range 11 {
		digitos[i], _ = strconv.Atoi(string(cpf[i]))
	}

	soma := 0
	for i := range 9 {
		soma += digitos[i] * (10 - i)
	}
	resto := soma % 11
	digito1 := 0
	if resto >= 2 {
		digito1 = 11 - resto
	}
	if digitos[9] != digito1 {
		return false
	}

	soma = 0
	for i := range 10 {
		soma += digitos[i] * (11 - i)
	}
	resto = soma % 11
	digito2 := 0
	if resto >= 2 {
		digito2 = 11 - resto
	}
	if digitos[10] != digito2 {
		return false
	}

	return true
}

func ValidateCEP(cep string) bool {
	// Remove caracteres não numéricos
	cep = regexp.MustCompile(`[^0-9]`).ReplaceAllString(cep, "")

	// Verifica se tem exatamente 8 dígitos
	if len(cep) != 8 {
		return false
	}

	// Verifica se todos os dígitos são iguais
	todosIguais := true
	for i := 1; i < 8; i++ {
		if cep[i] != cep[0] {
			todosIguais = false
			break
		}
	}

	return !todosIguais
}

func ValidateDate(data string) bool {
	pattern := `^(0[1-9]|[12][0-9]|3[01])/(0[1-9]|1[012])/(19|20)\d\d$`
	matched, _ := regexp.MatchString(pattern, data)
	if !matched {
		return false
	}

	parts := strings.Split(data, "/")
	if len(parts) != 3 {
		return false
	}

	day, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	year, _ := strconv.Atoi(parts[2])

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	return date.Year() == year && int(date.Month()) == month && date.Day() == day
}

func ValidateDateISO(data string) bool {
	pattern := `^(19|20)\d\d-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])$`
	matched, _ := regexp.MatchString(pattern, data)
	if !matched {
		return false
	}

	parts := strings.Split(data, "-")
	if len(parts) != 3 {
		return false
	}

	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])

	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	return date.Year() == year && int(date.Month()) == month && date.Day() == day
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
