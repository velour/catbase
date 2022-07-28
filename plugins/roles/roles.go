package roles

import (
	"fmt"
	"github.com/velour/catbase/bot/msg"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

type RolesPlugin struct {
	b  bot.Bot
	c  *config.Config
	db *sqlx.DB
	h  bot.HandlerTable
}

func New(b bot.Bot) *RolesPlugin {
	p := &RolesPlugin{
		b:  b,
		c:  b.Config(),
		db: b.DB(),
	}
	p.Register()
	return p
}

var rolesRegexp = regexp.MustCompile(`(?i)^lsroles\s?(?P<offering>.*)`)

func (p *RolesPlugin) Register() {
	p.h = bot.HandlerTable{
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^role (?P<role>.+)$`),
			HelpText: "Toggles a role on or off for you. Role must be of the current offerings",
			Handler:  p.toggleRole,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    rolesRegexp,
			HelpText: "Lists roles with an optional offering set specifier",
			Handler:  p.lsRoles,
		},
		{
			Kind: bot.Help, IsCmd: true,
			Regex:    regexp.MustCompile(`.`),
			HelpText: "Lists roles with an optional offering set specifier",
			Handler:  p.lsRoles,
		},
	}
	p.b.Register(p, bot.Help, func(c bot.Connector, k bot.Kind, m msg.Message, args ...any) bool {
		return p.lsRoles(bot.Request{
			Conn:   c,
			Kind:   k,
			Msg:    m,
			Values: bot.ParseValues(rolesRegexp, m.Body),
			Args:   args,
		})
	})
	p.b.RegisterTable(p, p.h)
}

func (p *RolesPlugin) toggleRole(r bot.Request) bool {
	role := r.Values["role"]
	offering := p.c.Get("roles.currentoffering", "")
	if !strings.HasSuffix(role, offering) {
		role = fmt.Sprintf("%s-%s", role, offering)
	}
	roles, err := r.Conn.GetRoles()
	if err != nil {
		log.Error().Err(err).Msg("getRoles")
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "I couldn't get the roles for some reason.")
		return true
	}
	for _, rr := range roles {
		if strings.ToLower(rr.Name) == strings.ToLower(role) {
			if err = r.Conn.SetRole(r.Msg.User.ID, rr.ID); err != nil {
				log.Error().Err(err).Msg("setRole")
				p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "I couldn't set that role.")
				return true
			}
			p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I have toggled your role, %s.", rr.Name))
			return true
		}
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "I couldn't find that role.")
	return true
}

func (p *RolesPlugin) lsRoles(r bot.Request) bool {
	offering := r.Values["offering"]
	if offering == "" {
		// This would be all if we hit the fallback
		offering = p.c.Get("roles.currentOffering", "")
	}
	roles, err := r.Conn.GetRoles()
	if err != nil {
		log.Error().Err(err).Msg("getRoles")
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "I couldn't get the roles for some reason.")
		return true
	}
	msg := "Available roles: "
	i := 0
	for _, r := range roles {
		if !strings.HasSuffix(r.Name, offering) || r.Name == "@everyone" {
			continue
		}
		if i > 0 {
			msg += ", "
		}
		msg += fmt.Sprintf("%s", strings.TrimSuffix(r.Name, "-"+offering))
		i++
	}
	msg += "\n\nUse `!role [rolename]` to toggle your membership in one of these roles."
	if i == 0 {
		msg = "I found no matching roles."
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
	return true
}

func (p *RolesPlugin) setOffering(r bot.Request) bool {
	if !p.b.CheckAdmin(r.Msg.User.Name) {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "You must be an admin to set an offering.")
		return true
	}
	offering := r.Values["offering"]
	err := p.c.Set("roles.currentOffering", offering)
	if err != nil {
		log.Error().Err(err).Msg("setOffering")
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Could not set that offering")
		return true
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("New offering set to %s", offering))
	return true
}
