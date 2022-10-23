package auths

type AuthEntity struct {
	Username string
	Password string
	Token    string
}

func (a *AuthEntity) HashPassword(pwd string) *AuthEntity {
	hash := "123"
	a.Password = hash
	return a
}
