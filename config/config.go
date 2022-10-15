// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	sqlite3 "github.com/mattn/go-sqlite3"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// Config stores any system-wide startup information that cannot be easily configured via
// the database
type Config struct {
	*sqlx.DB

	DBFile  string
	secrets map[string]Secret
}

// Secret is a config value that is loaded permanently and not ever displayed
type Secret struct {
	// Key is the key field of the table
	Key string `db:"key"`
	// Value represents the secret that must not be shared
	Value string `db:"value"`
}

// GetFloat64 returns the config value for a string key
// It will first look in the env vars for the key
// It will check the db for the key if an env DNE
// Finally, it will return a zero value if the key does not exist
// It will attempt to convert the value to a float64 if it exists
func (c *Config) GetFloat64(key string, fallback float64) float64 {
	f, err := strconv.ParseFloat(c.GetString(key, fmt.Sprintf("%f", fallback)), 64)
	if err != nil {
		return 0.0
	}
	return f
}

// GetInt64 returns the config value for a string key
// It will first look in the env vars for the key
// It will check the db for the key if an env DNE
// Finally, it will return a zero value if the key does not exist
// It will attempt to convert the value to an int if it exists
func (c *Config) GetInt64(key string, fallback int64) int64 {
	i, err := strconv.ParseInt(c.GetString(key, strconv.FormatInt(fallback, 10)), 10, 64)
	if err != nil {
		return 0
	}
	return i
}

// GetInt returns the config value for a string key
// It will first look in the env vars for the key
// It will check the db for the key if an env DNE
// Finally, it will return a zero value if the key does not exist
// It will attempt to convert the value to an int if it exists
func (c *Config) GetInt(key string, fallback int) int {
	i, err := strconv.Atoi(c.GetString(key, strconv.Itoa(fallback)))
	if err != nil {
		return 0
	}
	return i
}

// GetBool returns true or false for config key
// It will assume false for any string except "true"
func (c *Config) GetBool(key string, fallback bool) bool {
	val := c.GetString(key, strconv.FormatBool(fallback))
	return val == "true"
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
// It will check the db for the key if an env DNE
// Finally, it will return a zero value if the key does not exist
// It will convert the value to a string if it exists
func (c *Config) GetString(key, fallback string) string {
	key = strings.ToLower(key)
	if v, found := os.LookupEnv(envkey(key)); found {
		return v
	}
	if v, found := c.secrets[key]; found {
		return v.Value
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

func (c *Config) GetMap(key string, fallback map[string]string) map[string]string {
	content := c.Get(key, "")
	if content == "" {
		return fallback
	}
	vals := map[string]string{}
	err := json.Unmarshal([]byte(content), &vals)
	if err != nil {
		log.Error().Err(err).Msgf("Could not decode config for %s", key)
		return fallback
	}
	return vals
}

// GetArray returns the string slice config value for a string key
// It will first look in the env vars for the key with ;; separated values
// Look, I'm too lazy to do parsing to ensure that a comma is what the user meant
// It will check the db for the key if an env DNE
// Finally, it will return a zero value if the key does not exist
// This will do no conversion.
func (c *Config) GetArray(key string, fallback []string) []string {
	val := c.GetString(key, "")
	if val == "" {
		return fallback
	}
	return strings.Split(val, ";;")
}

func (c *Config) Unset(key string) error {
	q := `delete from config where key=?`
	tx, err := c.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(q, key)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// Set changes the value for a configuration in the database
// Note, this is always a string. Use the SetArray for an array helper
func (c *Config) Set(key, value string) error {
	key = strings.ToLower(key)
	value = strings.Trim(value, "`")
	q := `insert into config (key,value) values (?, ?)
			on conflict(key) do update set value=?;`
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

func (c *Config) RefreshSecrets() error {
	q := `select key, value from secrets`
	var secrets []Secret
	err := c.Select(&secrets, q)
	if err != nil {
		return err
	}
	secretMap := map[string]Secret{}
	for _, s := range secrets {
		secretMap[s.Key] = s
	}
	c.secrets = secretMap
	return nil
}

func (c *Config) GetAllSecrets() map[string]Secret {
	return c.secrets
}

func (c *Config) SecretKeys() []string {
	keys := []string{}
	for k := range c.secrets {
		keys = append(keys, k)
	}
	return keys
}

func (c *Config) setSecret(key, value string) error {
	q := `insert into secrets (key,value) values (?, ?)
			on conflict(key) do update set value=?;`
	_, err := c.Exec(q, key, value, value)
	if err != nil {
		log.Fatal().Err(err).Msgf("secret")
		return err
	}
	return c.RefreshSecrets()
}

// RegisterSecret creates a new secret
func (c *Config) RegisterSecret(key, value string) error {
	return c.setSecret(key, value)
}

// RemoveSecret deregisters a secret
func (c *Config) RemoveSecret(key string) error {
	q := `delete from secrets where key=?`
	_, err := c.Exec(q, key)
	if err != nil {
		return err
	}
	return c.RefreshSecrets()
}

func (c *Config) SetMap(key string, values map[string]string) error {
	b, err := json.Marshal(values)
	if err != nil {
		return err
	}
	return c.Set(key, string(b))
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
		DBFile:  dbpath,
		secrets: map[string]Secret{},
	}
	c.DB = sqlDB

	if _, err := c.Exec(`create table if not exists config (
		key string,
		value string,
		primary key (key)
	);`); err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize config")
	}

	if _, err := c.Exec(`create table if not exists secrets (
		key string,
		value string,
		primary key (key)
	);`); err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize secrets")
	}

	if err := c.RefreshSecrets(); err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize config")
	}

	log.Info().Msgf("catbase is running.")

	return &c
}
