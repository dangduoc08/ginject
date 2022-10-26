package auths

import (
	"fmt"

	"github.com/dangduoc08/gooh/common"
)

type Authenticator interface {
	Signin(string, string) string
}

type AuthProvider struct {
	Username string
	Password string
}

func (userProvider AuthProvider) New() common.Provider {
	return userProvider
}

func (userProvider *AuthProvider) Signin(u string, p string) string {
	fmt.Println("username", u, "password", userProvider.Password)
	return "true"
}
