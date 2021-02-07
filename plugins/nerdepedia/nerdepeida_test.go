// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package nerdepedia

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/plugins/cli"

	"github.com/stretchr/testify/assert"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

var body = []byte(`
<meta name="description" content="Refresher Reading was a recurring feature appearing in Star Wars Insider. 20 Things You Didn&#039;t Know About the Tantive IV 20 Things You Didn&#039;t Know About the Mos Eisley Cantina 20 Things You Didn&#039;t Know About the Massassi Temples"/>
<link rel="canonical" href="https://starwars.fandom.com/wiki/Refresher_Reading"/>`)

type MockClient struct {
	Status int
	Body   io.ReadCloser
	Err    error
}

func (cl MockClient) Do(req *http.Request) (*http.Response, error) {
	log.Debug().Msgf("Returning mock response")
	return &http.Response{
		StatusCode: cl.Status,
		Body:       cl.Body,
	}, cl.Err
}

func makeMessage(payload string) bot.Request {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return bot.Request{
		Conn: &cli.CliPlugin{},
		Kind: bot.Message,
		Msg: msg.Message{
			User:    &user.User{Name: "tester"},
			Channel: "test",
			Body:    payload,
			Command: isCmd,
		},
	}
}

func TestWars(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	client = MockClient{
		Status: http.StatusOK,
		Body:   ioutil.NopCloser(bytes.NewReader(body)),
		Err:    nil,
	}
	res := c.message(makeMessage("help me obi-wan"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
}
