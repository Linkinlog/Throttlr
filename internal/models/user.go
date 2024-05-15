package models

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	UserCtxKey  ModelCtxKey = "user"
	sessionName string      = "auth"
)

var (
	NoSessionUser    = errors.New("no user in session")
	FailedAssertUser = errors.New("failed to assert user from session")
)

func UserSignedIn(ctx context.Context) bool {
	if _, ok := ctx.Value(UserCtxKey).(User); ok {
		return true
	}
	return false
}

func UserFromCtx(ctx context.Context) User {
	if user, ok := ctx.Value(UserCtxKey).(User); ok {
		return user
	}
	return User{}
}

func RemoveUserFromSession(r *http.Request, w http.ResponseWriter, gs sessions.Store) error {
	session, err := gs.Get(r, sessionName)
	if err != nil {
		return err
	}
	delete(session.Values, UserCtxKey)
	err = session.Save(r, w)
	if err != nil {
		return err
	}
	return nil
}

func UserFromSession(r *http.Request, gs sessions.Store) (User, error) {
	sess, err := gs.Get(r, sessionName)
	if err != nil {
		return User{}, err
	}
	val, ok := sess.Values[UserCtxKey]
	if !ok {
		return User{}, NoSessionUser
	}
	usr, ok := val.(User)
	if !ok {
		return User{}, FailedAssertUser
	}

	return usr, nil
}

func NewUser() *User {
	return &User{}
}

type User struct {
	Id, Name, Email string
}

func (u *User) SetName(name string) *User {
	u.Name = name
	return u
}

func (u *User) SetEmail(email string) *User {
	u.Email = email
	return u
}

func (u *User) SetId(userId string) *User {
	u.Id = userId
	return u
}

func (u *User) SaveToSession(r *http.Request, w http.ResponseWriter, gs sessions.Store) error {
	sess, err := gs.Get(r, sessionName)
	if err != nil {
		return err
	}
	sess.Values[UserCtxKey] = u
	err = gs.Save(r, w, sess)
	if err != nil {
		return err
	}
	return nil
}
