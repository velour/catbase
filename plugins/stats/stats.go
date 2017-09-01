// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Stats contains the plugin that allows the bot record chat statistics
package stats

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

const (
	DayFormat      = "2006-01-02"
	HourFormat     = "2006-01-02-15"
	HourBucket     = "hour"
	UserBucket     = "user"
	SightingBucket = "sighting"
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
	// date formatted: DayFormat
	day string
	// category
	bucket string
	// specific unique individual
	key string
	val value
}

func mkDay() string {
	return time.Now().Format(DayFormat)
}

// The value type is here in the future growth case that we might want to put a
// struct of more interesting information into the DB
type value int

func (v value) Bytes() ([]byte, error) {
	b, err := json.Marshal(v)
	return b, err
}

func valueFromBytes(b []byte) (value, error) {
	var v value
	err := json.Unmarshal(b, &v)
	return v, err
}

type stats []stat

// mkStat converts raw data to a stat struct
// Expected a string representation of the date formatted: DayFormat
func mkStat(day string, bucket, key, val []byte) (stat, error) {
	v, err := valueFromBytes(val)
	if err != nil {
		log.Printf("mkStat: error getting value from bytes: %s", err)
		return stat{}, err
	}
	return stat{
		day:    day,
		bucket: string(bucket),
		key:    string(key),
		val:    v,
	}, nil
}

// Another future-proofing function I shouldn't have written
func (v value) add(other value) value {
	return v + other
}

func openDB(path string) (*bolt.DB, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		log.Printf("Couldn't open BoltDB for stats (%s): %s", path, err)
		return nil, err
	}
	return db, err
}

// statFromDB takes a location specification and returns the data at that path
// Expected a string representation of the date formatted: DayFormat
func statFromDB(path, day, bucket, key string) (stat, error) {
	db, err := openDB(path)
	if err != nil {
		return stat{}, err
	}
	defer db.Close()
	buk := []byte(bucket)
	k := []byte(key)

	tx, err := db.Begin(true)
	if err != nil {
		log.Println("statFromDB: Error beginning the Tx")
		return stat{}, err
	}
	defer tx.Rollback()

	d, err := tx.CreateBucketIfNotExists([]byte(day))
	if err != nil {
		log.Println("statFromDB: Error creating the bucket")
		return stat{}, err
	}
	b, err := d.CreateBucketIfNotExists(buk)
	if err != nil {
		log.Println("statFromDB: Error creating the bucket")
		return stat{}, err
	}

	v := b.Get(k)

	if err := tx.Commit(); err != nil {
		log.Println("statFromDB: Error commiting the Tx")
		return stat{}, err
	}

	if v == nil {
		return stat{day, bucket, key, 0}, nil
	}

	return mkStat(day, buk, k, v)
}

// toDB takes a stat and records it, adding to the value in the DB if necessary
func (s stats) toDB(path string) error {
	db, err := openDB(path)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, stat := range s {
		err = db.Update(func(tx *bolt.Tx) error {
			d, err := tx.CreateBucketIfNotExists([]byte(stat.day))
			if err != nil {
				log.Println("toDB: Error creating bucket")
				return err
			}
			b, err := d.CreateBucketIfNotExists([]byte(stat.bucket))
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
			if stat.key == "" {
				log.Println("Keys should not be empty")
				return nil
			}
			log.Printf("Putting value in: '%s' %b, %+v", stat.key, []byte(stat.key), stat)
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

func (p *StatsPlugin) serveQuery(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(p.bot.Config().Stats.DBPath)
	defer f.Close()
	if err != nil {
		log.Printf("Error opening DB for web service: %s", err)
		fmt.Fprintf(w, "Error opening DB")
		return
	}
	http.ServeContent(w, r, "stats.db", time.Now(), f)
}

func (p *StatsPlugin) RegisterWeb() *string {
	http.HandleFunc("/stats", p.serveQuery)
	tmp := "/stats"
	return &tmp
}

func (p *StatsPlugin) mkUserStat(message msg.Message) stats {
	return stats{stat{mkDay(), UserBucket, message.User.Name, 1}}
}

func (p *StatsPlugin) mkHourStat(message msg.Message) stats {
	hr := time.Now().Hour()
	return stats{stat{mkDay(), HourBucket, strconv.Itoa(hr), 1}}
}

func (p *StatsPlugin) mkSightingStat(message msg.Message) stats {
	stats := stats{}
	for _, name := range p.bot.Config().Stats.Sightings {
		if strings.Contains(message.Body, name+" sighting") {
			stats = append(stats, stat{mkDay(), SightingBucket, name, 1})
		}
	}
	return stats
}

func (p *StatsPlugin) mkChannelStat(message msg.Message) stats {
	return stats{stat{mkDay(), "channel", message.Channel, 1}}
}
