package auths

import (
	"fmt"

	"github.com/dangduoc08/gooh/core"
)

type Authenticator interface {
	Signin(string, string) string
}

type AuthProvider struct {
	Username string
	Password string
}

func (userProvider AuthProvider) Inject() core.Provider {
	return userProvider
}

func (userProvider *AuthProvider) Signin(u string, p string) string {
	fmt.Println("username", u, "password", userProvider.Password)
	return "true"
}
