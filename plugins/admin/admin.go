// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package admin

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

// This is a admin plugin to serve as an example and quick copy/paste for new plugins.

type AdminPlugin struct {
	bot bot.Bot
	db  *sqlx.DB
	cfg *config.Config

	quiet bool
}

// New creates a new AdminPlugin with the Plugin interface
func New(b bot.Bot) *AdminPlugin {
	p := &AdminPlugin{
		bot: b,
		db:  b.DB(),
		cfg: b.Config(),
	}

	p.mkDB()

	b.RegisterRegex(p, bot.Message, comeBackRegex, p.comeBackCmd)
	b.RegisterRegexCmd(p, bot.Message, shutupRegex, p.shutupCmd)
	b.RegisterRegexCmd(p, bot.Message, addBlacklistRegex, p.isAdmin(p.addBlacklistCmd))
	b.RegisterRegexCmd(p, bot.Message, rmBlacklistRegex, p.isAdmin(p.rmBlacklistCmd))
	b.RegisterRegexCmd(p, bot.Message, addWhitelistRegex, p.isAdmin(p.addWhitelistCmd))
	b.RegisterRegexCmd(p, bot.Message, rmWhitelistRegex, p.isAdmin(p.rmWhitelistCmd))
	b.RegisterRegexCmd(p, bot.Message, allWhitelistRegex, p.isAdmin(p.allWhitelistCmd))
	b.RegisterRegexCmd(p, bot.Message, allUnWhitelistRegex, p.isAdmin(p.allUnWhitelistCmd))
	b.RegisterRegexCmd(p, bot.Message, getWhitelistRegex, p.isAdmin(p.getWhitelistCmd))
	b.RegisterRegexCmd(p, bot.Message, getPluginsRegex, p.isAdmin(p.getPluginsCmd))
	b.RegisterRegexCmd(p, bot.Message, rebootRegex, p.isAdmin(p.rebootCmd))
	b.RegisterRegexCmd(p, bot.Message, passwordRegex, p.passwordCmd)
	b.RegisterRegex(p, bot.Message, variableSetRegex, p.variableSetCmd)
	b.RegisterRegex(p, bot.Message, variableUnSetRegex, p.variableUnSetCmd)
	b.RegisterRegexCmd(p, bot.Message, unsetConfigRegex, p.isAdmin(p.unsetConfigCmd))
	b.RegisterRegexCmd(p, bot.Message, setConfigRegex, p.isAdmin(p.setConfigCmd))
	b.RegisterRegexCmd(p, bot.Message, pushConfigRegex, p.isAdmin(p.pushConfigCmd))
	b.RegisterRegexCmd(p, bot.Message, setKeyConfigRegex, p.isAdmin(p.setKeyConfigCmd))
	b.RegisterRegexCmd(p, bot.Message, getConfigRegex, p.isAdmin(p.getConfigCmd))
	b.RegisterRegexCmd(p, bot.Message, setNickRegex, p.setNick)

	b.Register(p, bot.Help, p.help)
	p.registerWeb()

	return p
}

var forbiddenKeys = map[string]bool{
	"twitch.authorization": true,
	"twitch.clientid":      true,
	"untappd.token":        true,
	"slack.token":          true,
	"meme.memes":           true,
}

func (p *AdminPlugin) mkDB() {
	q := `create table if not exists apppass (
    	id integer primary key autoincrement,
        secret string not null,
        encoded_pass string not null,
        cost integer default 10
	)`
	p.db.MustExec(q)
}

var shutupRegex = regexp.MustCompile(`(?i)^shut up$`)
var comeBackRegex = regexp.MustCompile(`(?i)^come back$`)

var addBlacklistRegex = regexp.MustCompile(`(?i)disable plugin (?P<plugin>.*)$`)
var rmBlacklistRegex = regexp.MustCompile(`(?i)enable plugin (?P<plugin>.*)$`)

var addWhitelistRegex = regexp.MustCompile(`(?i)^whitelist plugin (?P<plugin>.*)$`)
var rmWhitelistRegex = regexp.MustCompile(`(?i)^unwhitelist plugin (?P<plugin>.*)$`)
var allWhitelistRegex = regexp.MustCompile(`(?i)^whitelist all$`)
var allUnWhitelistRegex = regexp.MustCompile(`(?i)^unwhitelist all$`)
var getWhitelistRegex = regexp.MustCompile(`(?i)^list whitelist$`)
var getPluginsRegex = regexp.MustCompile(`(?i)^list plugins$`)

var rebootRegex = regexp.MustCompile(`^reboot$`)
var passwordRegex = regexp.MustCompile(`(?i)^password$`)

var variableSetRegex = regexp.MustCompile(`(?i)^\$(?P<var>\S+)\s?=\s?(?P<value>\S+)$`)
var variableUnSetRegex = regexp.MustCompile(`(?i)^\$(?P<var>\S+)\s?!=\s?(?P<value>\S+)$`)

var unsetConfigRegex = regexp.MustCompile(`(?i)^unset (?P<key>\S+)$`)
var setConfigRegex = regexp.MustCompile(`(?i)^set (?P<key>\S+) (?P<value>.*)$`)
var pushConfigRegex = regexp.MustCompile(`(?i)^push (?P<key>\S+) (?P<value>.*)$`)
var setKeyConfigRegex = regexp.MustCompile(`(?i)^setkey (?P<key>\S+) (?P<name>\S+) (?P<value>.*)$`)
var getConfigRegex = regexp.MustCompile(`(?i)^get (?P<key>\S+)$`)
var setNickRegex = regexp.MustCompile(`(?i)^nick (?P<nick>.+)$`)

func (p *AdminPlugin) isAdmin(rh bot.ResponseHandler) bot.ResponseHandler {
	return func(r bot.Request) bool {
		if !p.bot.CheckAdmin(r.Msg.User.ID) {
			log.Debug().Msgf("User %s is not an admin", r.Msg.User.Name)
			return false
		}
		return rh(r)
	}
}

func (p *AdminPlugin) shutupCmd(r bot.Request) bool {
	dur := time.Duration(p.cfg.GetInt("quietDuration", 5)) * time.Minute
	log.Info().Msgf("Going to sleep for %v, %v", dur, time.Now().Add(dur))
	leaveMsg := p.bot.Config().Get("admin.shutup", "Okay. I'll be back later.")
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, leaveMsg)
	p.quiet = true
	p.bot.SetQuiet(true)
	go func() {
		select {
		case <-time.After(dur):
			p.quiet = false
			p.bot.SetQuiet(false)
			log.Info().Msg("Waking up from nap.")
			backMsg := p.bot.Config().Get("admin.backmsg", "I'm back, bitches.")
			p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, backMsg)
		}
	}()
	return true
}

func (p *AdminPlugin) comeBackCmd(r bot.Request) bool {
	p.quiet = false
	p.bot.SetQuiet(false)
	backMsg := p.bot.Config().Get("admin.comeback", "Okay, I'm back.")
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, backMsg)
	return true
}

func (p *AdminPlugin) addBlacklistCmd(r bot.Request) bool {
	plugin := r.Values["plugin"]
	if err := p.addBlacklist(r.Msg.Channel, plugin); err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I couldn't add that item: %s", err))
		log.Error().Err(err).Msgf("error adding blacklist item")
		return true
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("%s disabled. Use `!enable plugin %s` to re-enable it.", plugin, plugin))
	return true
}

func (p *AdminPlugin) rmBlacklistCmd(r bot.Request) bool {
	plugin := r.Values["plugin"]
	if err := p.rmBlacklist(r.Msg.Channel, plugin); err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I couldn't remove that item: %s", err))
		log.Error().Err(err).Msgf("error removing blacklist item")
		return true
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("%s enabled. Use `!disable plugin %s` to disable it.", plugin, plugin))
	return true
}

func (p *AdminPlugin) addWhitelistCmd(r bot.Request) bool {
	plugin := r.Values["plugin"]
	if err := p.addWhitelist(plugin); err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I couldn't whitelist that item: %s", err))
		log.Error().Err(err).Msgf("error adding whitelist item")
		return true
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("%s enabled. Use `!unwhitelist plugin %s` to disable it.", plugin, plugin))
	return true
}

func (p *AdminPlugin) rmWhitelistCmd(r bot.Request) bool {
	plugin := r.Values["plugin"]
	if err := p.rmWhitelist(plugin); err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I couldn't unwhitelist that item: %s", err))
		log.Error().Err(err).Msgf("error removing whitelist item")
		return true
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("%s disabled. Use `!whitelist plugin %s` to enable it.", plugin, plugin))
	return true
}

func (p *AdminPlugin) allWhitelistCmd(r bot.Request) bool {
	plugins := p.bot.GetPluginNames()
	for _, plugin := range plugins {
		if err := p.addWhitelist(plugin); err != nil {
			p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I couldn't whitelist that item: %s", err))
			log.Error().Err(err).Msgf("error adding whitelist item")
			return true
		}
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "Enabled all plugins")
	return true
}

func (p *AdminPlugin) allUnWhitelistCmd(r bot.Request) bool {
	plugins := p.bot.GetPluginNames()
	for _, plugin := range plugins {
		if plugin == "admin" {
			continue
		}
		if err := p.rmWhitelist(plugin); err != nil {
			p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I couldn't unwhitelist that item: %s", err))
			log.Error().Err(err).Msgf("error removing whitelist item")
			return true
		}
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "Disabled all plugins")
	return true
}

func (p *AdminPlugin) getWhitelistCmd(r bot.Request) bool {
	list := p.bot.GetWhitelist()
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Whitelist: %v", list))
	return true
}

func (p *AdminPlugin) getPluginsCmd(r bot.Request) bool {
	plugins := p.bot.GetPluginNames()
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Plugins: %v", plugins))
	return true
}

func (p *AdminPlugin) rebootCmd(r bot.Request) bool {
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "brb")
	log.Info().Msgf("Got reboot command")
	os.Exit(0)
	return true
}

func (p *AdminPlugin) passwordCmd(r bot.Request) bool {
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, p.bot.GetPassword())
	return true
}

func (p *AdminPlugin) variableSetCmd(r bot.Request) bool {
	variable := strings.ToLower(r.Values["var"])
	value := r.Values["value"]

	var count int64
	row := p.db.QueryRow(`select count(*) from variables where value = ?`, variable, value)
	err := row.Scan(&count)
	if err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "I'm broke and need attention in my variable creation code.")
		log.Error().Err(err)
		return true
	}

	if count > 0 {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "I've already got that one.")
	} else {
		_, err := p.db.Exec(`INSERT INTO variables (name, value) VALUES (?, ?)`, variable, value)
		if err != nil {
			p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "I'm broke and need attention in my variable creation code.")
			log.Error().Err(err)
			return true
		}
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "Added.")
	}
	return true
}

func (p *AdminPlugin) variableUnSetCmd(r bot.Request) bool {
	variable := strings.ToLower(r.Values["var"])
	value := r.Values["value"]

	_, err := p.db.Exec(`delete from variables where name=? and value=?`, variable, value)
	if err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "I'm broke and need attention in my variable creation code.")
		log.Error().Err(err)
	} else {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "Removed.")
	}

	return true
}

func (p *AdminPlugin) unsetConfigCmd(r bot.Request) bool {
	key := r.Values["key"]
	if forbiddenKeys[key] {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "You cannot access that key")
		return true
	}
	if err := p.cfg.Unset(key); err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Unset error: %s", err))
		return true
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Unset %s", key))
	return true
}

func (p *AdminPlugin) setConfigCmd(r bot.Request) bool {
	key, value := r.Values["key"], r.Values["value"]
	if forbiddenKeys[key] {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "You cannot access that key")
		return true
	}
	if err := p.cfg.Set(key, value); err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Set error: %s", err))
		return true
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Set %s", key))
	return true
}

func (p *AdminPlugin) pushConfigCmd(r bot.Request) bool {
	key, value := r.Values["key"], r.Values["value"]
	if forbiddenKeys[key] {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "You cannot access that key")
		return true
	}
	items := p.cfg.GetArray(key, []string{})
	items = append(items, value)
	if err := p.cfg.Set(key, strings.Join(items, ";;")); err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Set error: %s", err))
		return true
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Set %s", key))
	return true
}

func (p *AdminPlugin) setKeyConfigCmd(r bot.Request) bool {
	key, name, value := r.Values["key"], r.Values["name"], r.Values["value"]
	if forbiddenKeys[key] {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "You cannot access that key")
		return true
	}
	items := p.cfg.GetMap(key, map[string]string{})
	items[name] = value
	if err := p.cfg.SetMap(key, items); err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Set error: %s", err))
		return true
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Set %s", key))
	return true
}

func (p *AdminPlugin) getConfigCmd(r bot.Request) bool {
	key := r.Values["key"]
	if forbiddenKeys[key] {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "You cannot access that key")
		return true
	}
	v := p.cfg.Get(key, "<unknown>")
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("%s: %s", key, v))
	return true
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *AdminPlugin) help(conn bot.Connector, kind bot.Kind, m msg.Message, args ...any) bool {
	p.bot.Send(conn, bot.Message, m.Channel, "This does super secret things that you're not allowed to know about.")
	return true
}

func (p *AdminPlugin) addWhitelist(plugin string) error {
	return p.modList(`insert or replace into pluginWhitelist values (?)`, "", plugin)
}

func (p *AdminPlugin) rmWhitelist(plugin string) error {
	if plugin == "admin" {
		return fmt.Errorf("you cannot disable the admin plugin")
	}
	return p.modList(`delete from pluginWhitelist where name=?`, "", plugin)
}

func (p *AdminPlugin) addBlacklist(channel, plugin string) error {
	if plugin == "admin" {
		return fmt.Errorf("you cannot disable the admin plugin")
	}
	return p.modList(`insert or replace into pluginBlacklist values (?, ?)`, channel, plugin)
}

func (p *AdminPlugin) rmBlacklist(channel, plugin string) error {
	return p.modList(`delete from pluginBlacklist where channel=? and name=?`, channel, plugin)
}

func (p *AdminPlugin) modList(query, channel, plugin string) error {
	if channel == "" && plugin != "" {
		channel = plugin // hack
	}
	plugins := p.bot.GetPluginNames()
	for _, pp := range plugins {
		if pp == plugin {
			if _, err := p.db.Exec(query, channel, plugin); err != nil {
				return fmt.Errorf("%w", err)
			}
			err := p.bot.RefreshPluginWhitelist()
			if err != nil {
				return fmt.Errorf("%w", err)
			}
			err = p.bot.RefreshPluginBlacklist()
			if err != nil {
				return fmt.Errorf("%w", err)
			}
			return nil
		}
	}
	err := fmt.Errorf("unknown plugin named '%s'", plugin)
	return err
}

func (p *AdminPlugin) setNick(r bot.Request) bool {
	if needAdmin := p.cfg.GetInt("nick.needsadmin", 1); needAdmin == 1 && !p.bot.CheckAdmin(r.Msg.User.ID) {
		return false
	}
	nick := r.Values["nick"]
	if err := r.Conn.Nick(nick); err != nil {
		log.Error().Err(err).Msg("set nick")
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "I can't seem to set a new nick.")
		return true
	}
	p.cfg.Set("nick", nick)
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I shall now be known as %s.", nick))
	return true
}
