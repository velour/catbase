// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package config

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

// Config stores any system-wide startup information that cannot be easily configured via
// the database
type Config struct {
	*sqlx.DB

	DBFile string
}

// GetFloat64 returns the config value for a string key
// It will first look in the env vars for the key
// It will check the DB for the key if an env DNE
// Finally, it will return a zero value if the key does not exist
// It will attempt to convert the value to a float64 if it exists
func (c *Config) GetFloat64(key string, fallback float64) float64 {
	f, err := strconv.ParseFloat(c.GetString(key, fmt.Sprintf("%f", fallback)), 64)
	if err != nil {
		return 0.0
	}
	return f
}

// GetInt returns the config value for a string key
// It will first look in the env vars for the key
// It will check the DB for the key if an env DNE
// Finally, it will return a zero value if the key does not exist
// It will attempt to convert the value to an int if it exists
func (c *Config) GetInt(key string, fallback int) int {
	i, err := strconv.Atoi(c.GetString(key, strconv.Itoa(fallback)))
	if err != nil {
		return 0
	}
	return i
}

// Get is a shortcut for GetString
func (c *Config) Get(key, fallback string) string {
	return c.GetString(key, fallback)
}

func envkey(key string) string {
	key = strings.ToUpper(key)
	key = strings.Replace(key, ".", "", -1)
	return key
}

// GetString returns the config value for a string key
// It will first look in the env vars for the key
// It will check the DB for the key if an env DNE
// Finally, it will return a zero value if the key does not exist
// It will convert the value to a string if it exists
func (c *Config) GetString(key, fallback string) string {
	key = strings.ToLower(key)
	if v, found := os.LookupEnv(envkey(key)); found {
		return v
	}
	var configValue string
	q := `select value from config where key=?`
	err := c.DB.Get(&configValue, q, key)
	if err != nil {
		log.Debug().Msgf("WARN: Key %s is empty", key)
		return fallback
	}
	return configValue
}

// GetArray returns the string slice config value for a string key
// It will first look in the env vars for the key with ;; separated values
// Look, I'm too lazy to do parsing to ensure that a comma is what the user meant
// It will check the DB for the key if an env DNE
// Finally, it will return a zero value if the key does not exist
// This will do no conversion.
func (c *Config) GetArray(key string, fallback []string) []string {
	val := c.GetString(key, "")
	if val == "" {
		return fallback
	}
	return strings.Split(val, ";;")
}

// Set changes the value for a configuration in the database
// Note, this is always a string. Use the SetArray for an array helper
func (c *Config) Set(key, value string) error {
	key = strings.ToLower(key)
	q := (`insert into config (key,value) values (?, ?)
		on conflict(key) do update set value=?;`)
	tx, err := c.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(q, key, value, value)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) SetArray(key string, values []string) error {
	vals := strings.Join(values, ";;")
	return c.Set(key, vals)
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

// Readconfig loads the config data out of a JSON file located in cfile
func ReadConfig(dbpath string) *Config {
	if dbpath == "" {
		dbpath = "catbase.db"
	}
	log.Info().Msgf("Using %s as database file.\n", dbpath)

	sqlDB, err := sqlx.Open("sqlite3_custom", dbpath)
	if err != nil {
		log.Fatal().Err(err)
	}
	c := Config{
		DBFile: dbpath,
	}
	c.DB = sqlDB

	if _, err := c.Exec(`create table if not exists config (
		key string,
		value string,
		primary key (key)
	);`); err != nil {
		panic(err)
	}

	log.Info().Msgf("catbase is running.")

	return &c
}
