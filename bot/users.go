// © 2013 the AlePale Authors under the WTFPL. See AUTHORS for the list of authors.

package bot

// User type stores user history. This is a vehicle that will follow the user for the active
// session
type User struct {
	// Current nickname known
	Name string

	// LastSeen	DateTime

	// Alternative nicknames seen
	Alts   []string
	Parent string

	Admin bool

	//bot *Bot
}

var users map[string]*User

func (b *Bot) GetUser(nick string) *User {
	if _, ok := users[nick]; !ok {
		users[nick] = &User{
			Name:  nick,
			Admin: b.checkAdmin(nick),
		}
	}
	return users[nick]
}

func (b *Bot) NewUser(nick string) *User {
	return &User{
		Name:  nick,
		Admin: b.checkAdmin(nick),
	}
}

func (b *Bot) checkAdmin(nick string) bool {
	for _, u := range b.Config.Admins {
		if nick == u {
			return true
		}
	}
	return false
}
