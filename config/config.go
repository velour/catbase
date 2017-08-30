// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"
)

// Config stores any system-wide startup information that cannot be easily configured via
// the database
type Config struct {
	DBConn *sqlx.DB

	DB struct {
		File   string
		Name   string
		Server string
	}
	Channels    []string
	MainChannel string
	Plugins     []string
	Type        string
	Irc         struct {
		Server, Pass string
	}
	Slack struct {
		Token string
	}
	Nick        string
	FullName    string
	Version     string
	CommandChar []string
	RatePerSec  float64
	LogLength   int
	Admins      []string
	HttpAddr    string
	Untappd     struct {
		Token    string
		Freq     int
		Channels []string
	}
	Twitch struct {
		Freq  int
		Users map[string][]string //channel -> usernames
	}
	EnforceNicks          bool
	WelcomeMsgs           []string
	TwitterConsumerKey    string
	TwitterConsumerSecret string
	TwitterUserKey        string
	TwitterUserSecret     string
	BadMsgs               []string
	Bad                   struct {
		Msgs  []string
		Nicks []string
		Hosts []string
	}
	Your struct {
		MaxLength     int
		Replacements []Replacement
	}
	LeftPad struct {
		MaxLen int
		Who    string
	}
	Factoid struct {
		MinLen      int
		QuoteChance float64
		QuoteTime   int
		StartupFact string
	}
	Babbler struct {
		DefaultUsers []string
	}
	Reminder struct {
		MaxBatchAdd int
	}
	Stats struct {
		DBPath    string
		Sightings []string
	}
	Emojify struct {
		Chance float64
	}
	Reaction struct {
		GeneralChance float64
		HarrassChance float64
		HarrassList []string
		PositiveReactions []string
		NegativeReactions []string
	}
}

func init() {
	regex := func(re, s string) (bool, error) {
		return regexp.MatchString(re, s)
	}
	sql.Register("sqlite3_custom",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("REGEXP", regex, true)
			},
		})
}

type Replacement struct {
	This string
	That string
	Frequency float64
}

// Readconfig loads the config data out of a JSON file located in cfile
func Readconfig(version, cfile string) *Config {
	fmt.Printf("Using %s as config file.\n", cfile)
	file, e := ioutil.ReadFile(cfile)
	if e != nil {
		panic("Couldn't read config file!")
	}

	var c Config
	err := json.Unmarshal(file, &c)
	if err != nil {
		panic(err)
	}
	c.Version = version

	if c.Type == "" {
		c.Type = "irc"
	}

	fmt.Printf("godeepintir version %s running.\n", c.Version)

	sqlDB, err := sqlx.Open("sqlite3_custom", c.DB.File)
	if err != nil {
		log.Fatal(err)
	}
	c.DBConn = sqlDB

	return &c
}
