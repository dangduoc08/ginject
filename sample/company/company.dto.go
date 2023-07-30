package company

import (
	"regexp"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/exception"
)

type CreateCompanyBody struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

func isValidEmail(email string) bool {
	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	regex := regexp.MustCompile(emailPattern)

	return regex.MatchString(email)
}

func (dto CreateCompanyBody) Transform(body gooh.Body, metadata common.ArgumentMetadata) any {
	if v, ok := body["name"]; !ok || v.(string) == "" {
		panic(exception.UnprocessableEntityException("Invalid name"))
	} else {
		dto.Name = v.(string)
	}

	if v, ok := body["email"]; !ok || !isValidEmail(v.(string)) {
		panic(exception.UnprocessableEntityException("Invalid email"))
	} else {
		dto.Email = v.(string)
	}

	return dto
}
