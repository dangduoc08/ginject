package accounts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/crypto"
	"github.com/dangduoc08/ginject/modules/cache"
	dbstorage "github.com/dangduoc08/ginject/modules/storage"
)

const (
	usersTable     = "users"
	sessionTTL     = 24 * time.Hour
	sessionKeyPath = "session:"
)

type UserService struct {
	Store dbstorage.StoreService
	Cache cache.CacheService
}

func (svc UserService) NewProvider() core.Provider {
	svc.users().Schema(dbstorage.ModelSchema{
		Fields: []dbstorage.FieldSchema{
			{Name: "email", Index: true},
		},
	})

	return svc
}

func (svc UserService) users() *dbstorage.Model {
	return svc.Store.Model(usersTable)
}

func (svc UserService) Register(email, name, password string) (User, error) {
	existing, err := svc.users().Find().Where("email", dbstorage.OpEq, email).Exec()
	if err != nil {
		return User{}, exception.InternalServerErrorException("failed to look up email")
	}
	if len(existing) > 0 {
		return User{}, exception.ConflictException("email is already registered")
	}

	doc, err := svc.users().Create(map[string]any{
		"email":        email,
		"name":         name,
		"passwordHash": hashPassword(password),
	})
	if err != nil {
		return User{}, exception.InternalServerErrorException("failed to create user")
	}

	return userFromDocument(doc), nil
}

func (svc UserService) Authenticate(email, password string) (User, string, error) {
	docs, err := svc.users().Find().Where("email", dbstorage.OpEq, email).Exec()
	if err != nil || len(docs) == 0 {
		return User{}, "", exception.UnauthorizedException("invalid email or password")
	}

	doc := docs[0]
	passwordHash, _ := doc.Data["passwordHash"].(string)
	if passwordHash != hashPassword(password) {
		return User{}, "", exception.UnauthorizedException("invalid email or password")
	}

	token, err := crypto.UUID()
	if err != nil {
		return User{}, "", exception.InternalServerErrorException("failed to start session")
	}

	if err := svc.Cache.Set(context.Background(), sessionKey(token), []byte(doc.ID), sessionTTL); err != nil {
		return User{}, "", exception.InternalServerErrorException("failed to start session")
	}

	return userFromDocument(doc), token, nil
}

func (svc UserService) Logout(token string) {
	_ = svc.Cache.Delete(context.Background(), sessionKey(token))
}

func (svc UserService) UserBySession(token string) (User, bool) {
	userID, ok := svc.Cache.Get(context.Background(), sessionKey(token))
	if !ok {
		return User{}, false
	}

	doc, err := svc.users().FindByID(string(userID))
	if err != nil {
		return User{}, false
	}

	return userFromDocument(doc), true
}

func sessionKey(token string) string {
	return sessionKeyPath + token
}

func userFromDocument(doc dbstorage.Document) User {
	email, _ := doc.Data["email"].(string)
	name, _ := doc.Data["name"].(string)

	return User{
		ID:    doc.ID,
		Email: email,
		Name:  name,
	}
}

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}
