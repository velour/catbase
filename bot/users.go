package bot

import (
	// "labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
)

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

	bot *Bot
}

func NewUser(nick string) *User {
	return &User{
		Name:  nick,
		Admin: false,
	}
}

func (b *Bot) GetUser(nick string) *User {
	coll := b.Db.C("users")
	query := coll.Find(bson.M{"nick": nick})
	var user *User

	if count, err := query.Count(); err != nil {
		log.Printf("Error fetching user, %s: %s\n", nick, err)
		user = NewUser(nick)
		coll.Insert(NewUser(nick))
	} else if count == 1 {
		query.One(user)
	} else if count == 0 {
		// create the user
		user = NewUser(nick)
		coll.Insert(NewUser(nick))
	} else {
		log.Printf("Error: %s appears to have more than one user?\n", nick)
		query.One(user)
	}

	// grab linked user, if any
	if user.Parent != "" {
		query := coll.Find(bson.M{"Name": user.Parent})
		if count, err := query.Count(); err != nil && count == 1 {
			query.One(user)
		} else {
			log.Printf("Error: bad linkage on %s -> %s.\n",
				user.Name,
				user.Parent)
		}
	}

	user.bot = b

	found := false
	for _, u := range b.Users {
		if u.Name == user.Name {
			found = true
		}
	}
	if !found {
		b.Users = append(b.Users, *user)
	}

	return user
}

// Modify user entry to be a link to other, return other
func (u *User) LinkUser(other string) *User {
	coll := u.bot.Db.C("users")
	user := u.bot.GetUser(u.Name)
	otherUser := u.bot.GetUser(other)

	otherUser.Alts = append(otherUser.Alts, user.Alts...)
	user.Alts = []string{}
	user.Parent = other

	err := coll.Update(bson.M{"Name": u.Name}, u)
	if err != nil {
		log.Printf("Error updating user: %s\n", u.Name)
	}

	err = coll.Update(bson.M{"Name": other}, otherUser)
	if err != nil {
		log.Printf("Error updating other user: %s\n", other)
	}

	return otherUser
}

func (b *Bot) checkAdmin(nick string) bool {
	for _, u := range b.Config.Admins {
		if nick == u {
			return true
		}
	}
	return false
}
