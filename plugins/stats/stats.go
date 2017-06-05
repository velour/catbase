// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Stats contains the plugin that allows the bot record chat statistics
package stats

import (
	"bytes"
	"encoding/gob"
	"log"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type StatsPlugin struct {
	bot    bot.Bot
	config *config.Config
}

// New creates a new StatsPlugin with the Plugin interface
func New(bot bot.Bot) *StatsPlugin {
	p := StatsPlugin{
		bot:    bot,
		config: bot.Config(),
	}
	return &p

}

type stat struct {
	bucket string
	key    string
	val    value
}

// The value type is here in the future growth case that we might want to put a
// struct of more interesting information into the DB
type value int

func (v value) Bytes() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	err := enc.Encode(v)
	return b.Bytes(), err
}

func valueFromBytes(b []byte) (value, error) {
	var v value
	buf := bytes.NewReader(b)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&v)
	return v, err
}

type stats []stat

func mkStat(bucket, key, val []byte) (stat, error) {
	v, err := valueFromBytes(val)
	if err != nil {
		log.Printf("mkStat: error getting value from bytes: %s", err)
		return stat{}, err
	}
	return stat{
		bucket: string(bucket),
		key:    string(key),
		val:    v,
	}, nil
}

// Another future-proofing function I shouldn't have written
func (v value) add(other value) value {
	return v + other
}

func statFromDB(path, bucket, key string) (stat, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	buk := []byte(bucket)
	k := []byte(key)
	if err != nil {
		log.Printf("Couldn't open BoltDB for stats: %s", err)
		return stat{}, err
	}
	defer db.Close()

	tx, err := db.Begin(true)
	if err != nil {
		log.Println("statFromDB: Error beginning the Tx")
		return stat{}, err
	}
	defer tx.Rollback()

	b, err := tx.CreateBucketIfNotExists(buk)
	if err != nil {
		log.Println("statFromDB: Error creating the bucket")
		return stat{}, err
	}
	v := b.Get(k)

	if err := tx.Commit(); err != nil {
		log.Println("statFromDB: Error commiting the Tx")
		return stat{}, err
	}

	return mkStat(buk, k, v)
}

func (s stats) toDB(path string) error {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Printf("Couldn't open BoltDB for stats: %s", err)
		return err
	}
	defer db.Close()

	for _, stat := range s {
		err = db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte(stat.bucket))
			if err != nil {
				log.Println("toDB: Error creating bucket")
				return err
			}

			valueInDB := b.Get([]byte(stat.key))
			if valueInDB != nil {
				val, err := valueFromBytes(valueInDB)
				if err != nil {
					log.Println("toDB: Error getting value from bytes")
					return err
				}
				stat.val = stat.val.add(val)
			}

			v, err := stat.val.Bytes()
			if err != nil {
				return err
			}
			err = b.Put([]byte(stat.key), v)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *StatsPlugin) record(message msg.Message) {
	statGenerators := []func(msg.Message) stats{
		p.mkUserStat,
		p.mkHourStat,
		p.mkChannelStat,
		p.mkSightingStat,
	}

	allStats := stats{}

	for _, mk := range statGenerators {
		stats := mk(message)
		if stats != nil {
			allStats = append(allStats, stats...)
		}
	}

	allStats.toDB(p.bot.Config().Stats.DBPath)
}

func (p *StatsPlugin) Message(message msg.Message) bool {
	p.record(message)
	return false
}

func (p *StatsPlugin) Event(e string, message msg.Message) bool {
	p.record(message)
	return false
}

func (p *StatsPlugin) BotMessage(message msg.Message) bool {
	p.record(message)
	return false
}

func (p *StatsPlugin) Help(e string, m []string) {
}

func (p *StatsPlugin) RegisterWeb() *string {
	return nil
}

func (p *StatsPlugin) mkUserStat(message msg.Message) stats {
	return stats{stat{"user", message.User.Name, 1}}
}

func (p *StatsPlugin) mkHourStat(message msg.Message) stats {
	hr := time.Now().Hour()
	return stats{stat{"user", string(hr), 1}}
}

func (p *StatsPlugin) mkSightingStat(message msg.Message) stats {
	stats := stats{}
	for _, name := range p.bot.Config().Stats.Sightings {
		if strings.Contains(message.Body, name+" sighting") {
			stats = append(stats, stat{"sighting", name, 1})
		}
	}
	return stats
}

func (p *StatsPlugin) mkChannelStat(message msg.Message) stats {
	return stats{stat{"channel", message.Channel, 1}}
}
