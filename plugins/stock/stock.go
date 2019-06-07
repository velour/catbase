package stock

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type StockPlugin struct {
	bot    bot.Bot
	apiKey string
}

func New(b bot.Bot) *StockPlugin {
	s := &StockPlugin{
		bot:    b,
		apiKey: b.Config().GetString("Stock.API_KEY", "0E1DP61SJ7GF81IE"),
	}
	b.Register(s, bot.Message, s.message)
	b.Register(s, bot.Help, s.help)
	return s
}

type GlobalQuote struct {
	Info StockInfo `json:"GlobalQuote"`
}

type StockInfo struct {
	Symbol           string `json:"symbol"`
	Open             string `json:"open"`
	High             string `json:"high"`
	Low              string `json:"low"`
	Price            string `json:"price"`
	Volume           string `json:"volume"`
	LatestTradingDay string `json:"latesttradingday"`
	PreviousClose    string `json:"previousclose"`
	Change           string `json:"change"`
	ChangePercent    string `json:"changepercent"`
}

func (p *StockPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if !message.Command {
		return false
	}

	tokens := strings.Fields(message.Body)
	numTokens := len(tokens)

	if numTokens == 2 && strings.ToLower(tokens[0]) == "stock-price" {
		query := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", tokens[1], p.apiKey)

		resp, err := http.Get(query)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get stock info")
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal().Err(err).Msg("Error stock info body")
		}

		response := "Failed to retrieve data for stock symbol: " + tokens[1]

		cleaned := strings.ReplaceAll(string(body), " ", "")
		regex := regexp.MustCompile("[0-9][0-9]\\.")

		cleaned = regex.ReplaceAllString(cleaned, "")

		var info GlobalQuote
		err = json.Unmarshal([]byte(cleaned), &info)

		if err == nil && strings.EqualFold(tokens[1], info.Info.Symbol) {
			response = fmt.Sprintf("%s : $%s (%s)", tokens[1], info.Info.Price, info.Info.ChangePercent)
		}

		p.bot.Send(c, bot.Message, message.Channel, response)
		return true
	}

	return false
}

func (p *StockPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.bot.Send(c, bot.Message, message.Channel, "try '!stock-price SYMBOL'")
	return true
}
