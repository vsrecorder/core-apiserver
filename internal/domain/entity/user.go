package entity

type User struct {
	ID          string
	DisplayName string
	PhotoURL    string
}

func NewUser(
	id string,
	displayName string,
	photoURL string,
) *User {
	return &User{
		ID:          id,
		DisplayName: displayName,
		PhotoURL:    photoURL,
	}
}
