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

// UserService is an account directory and session store. Accounts persist to
// disk through the storage module; sessions live in the cache module behind a
// bearer token, so they expire automatically after sessionTTL.
//
// It exists to demonstrate the framework's DI, persistence and caching
// modules; SHA-256 password hashing is illustrative only and not suitable for
// production use.
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

// Register creates a new account. It returns a ConflictException if the
// email is already taken.
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

// Authenticate verifies an email/password pair and starts a new session,
// returning the user and a bearer token. It returns an
// UnauthorizedException when the credentials don't match.
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

// Logout ends the session identified by token. Unknown tokens are ignored,
// so logout is idempotent.
func (svc UserService) Logout(token string) {
	_ = svc.Cache.Delete(context.Background(), sessionKey(token))
}

// UserBySession resolves a bearer token to the user that owns it.
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
