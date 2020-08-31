package discord

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/velour/catbase/bot/msg"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

type Discord struct {
	config *config.Config
	client *discordgo.Session

	event bot.Callback

	emojiCache map[string]string
}

func New(config *config.Config) *Discord {
	client, err := discordgo.New("Bot " + config.Get("DISCORDBOTTOKEN", ""))
	if err != nil {
		log.Fatal().Err(err).Msg("Could not connect to Discord")
	}
	d := &Discord{
		config: config,
		client: client,
	}
	return d
}

func (d *Discord) RegisterEvent(callback bot.Callback) {
	d.event = callback
}

func (d Discord) Send(kind bot.Kind, args ...interface{}) (string, error) {

	switch kind {
	case bot.Message:
		st, err := d.client.ChannelMessageSend(args[0].(string), args[1].(string))
		return st.ID, err
	default:
		log.Error().Msgf("discord.Send: unknown kind, %+v", kind)
		return "", errors.New("unknown message type")
	}
}

func (d *Discord) GetEmojiList() map[string]string {
	if d.emojiCache != nil {
		return d.emojiCache
	}
	guidID := d.config.Get("discord.guildid", "")
	if guidID == "" {
	}
	e, err := d.client.GuildEmojis(guidID)
	if err != nil {
		log.Error().Err(err).Msg("could not retrieve emojis")
		return map[string]string{}
	}
	emojis := map[string]string{}
	for _, e := range e {
		emojis[e.ID] = e.Name
	}
	return emojis
}

func (d *Discord) Who(id string) []string {
	ch, err := d.client.Channel(id)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting users")
		return []string{}
	}
	users := []string{}
	for _, u := range ch.Recipients {
		users = append(users, u.Username)
	}
	return users
}

func (d *Discord) Profile(id string) (user.User, error) {
	u, err := d.client.User(id)
	if err != nil {
		log.Error().Err(err).Msg("Error getting user")
		return user.User{}, err
	}
	return *convertUser(u), nil
}

func convertUser(u *discordgo.User) *user.User {
	return &user.User{
		ID:    u.ID,
		Name:  u.Username,
		Admin: false,
		Icon:  u.Avatar,
	}
}

func (d *Discord) Serve() error {
	log.Debug().Msg("starting discord serve function")

	d.client.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages)

	err := d.client.Open()
	if err != nil {
		log.Debug().Err(err).Msg("error opening client")
		return err
	}

	log.Debug().Msg("discord connection open")

	d.client.AddHandler(d.messageCreate)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	log.Debug().Msg("closing discord connection")

	// Cleanly close down the Discord session.
	return d.client.Close()
}

func (d *Discord) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Debug().Msgf("discord message: %+v", m)
	if m.Author.ID == s.State.User.ID {
		return
	}

	msg := msg.Message{
		User:        convertUser(m.Author),
		Channel:     m.ChannelID,
		ChannelName: m.ChannelID,
		Body:        m.Content,
		Time:        time.Now(),
	}

	d.event(d, bot.Message, msg)
}
