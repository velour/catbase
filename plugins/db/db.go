package db

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type DBPlugin struct {
	bot    bot.Bot
	config *config.Config
}

func New(b bot.Bot) *DBPlugin {
	return &DBPlugin{b, b.Config()}
}

func (p *DBPlugin) Message(message msg.Message) bool            { return false }
func (p *DBPlugin) Event(kind string, message msg.Message) bool { return false }
func (p *DBPlugin) ReplyMessage(msg.Message, string) bool       { return false }
func (p *DBPlugin) BotMessage(message msg.Message) bool         { return false }
func (p *DBPlugin) Help(channel string, parts []string)         {}

func (p *DBPlugin) RegisterWeb() *string {
	http.HandleFunc("/db/catbase.db", p.serveQuery)
	tmp := "/db/catbase.db"
	return &tmp
}

func (p *DBPlugin) serveQuery(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(p.bot.Config().DB.File)
	defer f.Close()
	if err != nil {
		log.Printf("Error opening DB for web service: %s", err)
		fmt.Fprintf(w, "Error opening DB")
		return
	}
	http.ServeContent(w, r, "catbase.db", time.Now(), f)
}
