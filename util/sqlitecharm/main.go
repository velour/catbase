package main

import (
	"flag"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/plugins/admin"
)

var (
	dbPath  = flag.String("db", "catbase.db", "path to sqlite3 database")
	kvSpace = flag.String("kv", "catbase", "Namespace to use for charm store")
)

func main() {
	flag.Parse()

	c := config.ReadConfig(*dbPath, *kvSpace)

	cfgs := allConfigs(c.DB)
	for _, cfg := range cfgs {
		log.Debug().Msgf("Saving key %s", cfg.Key)
		err := c.KV.Set(cfg.Key, cfg.Value)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to set charm config")
		}
	}
}

func allConfigs(db *sqlx.DB) []admin.ConfigEntry {
	var configEntries []admin.ConfigEntry
	q := `select key, value from config`
	err := db.Select(&configEntries, q)
	if err != nil {
		log.Fatal().Err(err).Msgf("error getting configs")
	}

	return configEntries
}
