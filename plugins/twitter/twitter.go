package twitter

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ChimeraCoder/anaconda"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

type Twitter struct {
	c   *config.Config
	b   bot.Bot
	api *anaconda.TwitterApi
}

func New(b bot.Bot) *Twitter {
	t := Twitter{
		c: b.Config(),
		b: b,
	}
	token := t.c.GetString("twitter.accesstoken", "")
	secret := t.c.GetString("twitter.accesssecret", "")
	consumerKey := t.c.GetString("twitter.consumerkey", "")
	consumerSecret := t.c.GetString("twitter.consumersecret", "")
	//delay := t.c.GetInt("twitter.delay", 10)

	// Don't create the API or start the loop without credentials
	if token == "" || secret == "" || consumerKey == "" || consumerSecret == "" {
		log.Error().
			Str("token", token).
			Str("secret", secret).
			Str("consumerKey", consumerKey).
			Str("consumerSecret", consumerSecret).
			Msg("Could not start Twitter API")
		return &t
	}

	t.api = anaconda.NewTwitterApiWithCredentials(token, secret, consumerKey, consumerSecret)
	go t.timer(b.DefaultConnector())

	return &t
}

func (t *Twitter) timer(c bot.Connector) {
	checkTime := t.c.GetInt("twitter.checktime", 10)
	ticker := time.NewTicker(time.Duration(checkTime) * time.Minute)
	t.check(c)
	for {
		<-ticker.C
		t.check(c)
	}
}

func (t *Twitter) check(c bot.Connector) {
	log.Debug().Msg("checking twitter")
	chs := t.c.GetArray("twitter.channels", []string{})
	users := t.c.GetArray("twitter.users", []string{})
	for _, u := range users {
		vals := url.Values{"screen_name": []string{u}}
		tweets, err := t.api.GetUserTimeline(vals)
		if err != nil {
			log.Error().Err(err).Str("user", u).Msg("error getting tweets")
			continue
		}
		for _, tweet := range tweets {
			userKey := fmt.Sprintf("twitter.last.%s", u)
			lastTweet := int64(t.c.GetInt(userKey, 0))
			if lastTweet < tweet.Id {
				link := fmt.Sprintf("https://twitter.com/%s/status/%s", u, tweet.IdStr)
				log.Debug().Str("tweet", link).Msg("Unknown tweet")
				for _, ch := range chs {
					log.Debug().Str("ch", ch).Msg("Sending tweet")
					t.b.Send(c, bot.Message, ch, link)
				}
				t.c.Set(userKey, strconv.FormatInt(tweet.Id, 10))
			}
		}
	}
}
