package models

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
