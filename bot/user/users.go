// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package user

import "image"

// User type stores user history. This is a vehicle that will follow the user for the active
// session
type User struct {
	// Current nickname known
	ID      string
	Name    string
	Admin   bool
	Icon    string
	IconImg image.Image
}

func New(name string) User {
	return User{
		Name: name,
	}
}
