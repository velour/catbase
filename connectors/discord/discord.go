package discord

import (
	"errors"
	"fmt"
	"strings"

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
		return d.sendMessage(args[0].(string), args[1].(string), false, args...)
	case bot.Action:
		return d.sendMessage(args[0].(string), args[1].(string), true, args...)
	case bot.Edit:
		st, err := d.client.ChannelMessageEdit(args[0].(string), args[1].(string), args[2].(string))
		return st.ID, err
	case bot.Reply:
		original, err := d.client.ChannelMessage(args[0].(string), args[1].(string))
		msg := args[2].(string)
		if err != nil {
			log.Error().Err(err).Msg("could not get original")
		} else {
			msg = fmt.Sprintf("> %s\n%s", original, msg)
		}
		return d.sendMessage(args[0].(string), msg, false, args...)
	case bot.Reaction:
		msg := args[2].(msg.Message)
		err := d.client.MessageReactionAdd(args[0].(string), msg.ID, args[1].(string))
		return args[1].(string), err
	case bot.Delete:
		ch := args[0].(string)
		id := args[1].(string)
		err := d.client.ChannelMessageDelete(ch, id)
		if err != nil {
			log.Error().Err(err).Msg("cannot delete message")
		}
		return id, err
	default:
		log.Error().Msgf("discord.Send: unknown kind, %+v", kind)
		return "", errors.New("unknown message type")
	}
}

func (d *Discord) sendMessage(channel, message string, meMessage bool, args ...interface{}) (string, error) {
	if meMessage && !strings.HasPrefix(message, "_") && !strings.HasSuffix(message, "_") {
		message = "_" + message + "_"
	}

	var embeds *discordgo.MessageEmbed

	for _, arg := range args {
		switch a := arg.(type) {
		case bot.ImageAttachment:
			//embeds.URL = a.URL
			embeds = &discordgo.MessageEmbed{}
			embeds.Description = a.AltTxt
			embeds.Image = &discordgo.MessageEmbedImage{
				URL:    a.URL,
				Width:  a.Width,
				Height: a.Height,
			}
		}
	}

	data := &discordgo.MessageSend{
		Content: message,
		Embed:   embeds,
	}

	log.Debug().
		Interface("data", data).
		Interface("args", args).
		Msg("sending message")

	st, err := d.client.ChannelMessageSendComplex(channel, data)

	//st, err := d.client.ChannelMessageSend(channel, message)
	if err != nil {
		log.Error().Err(err).Msg("Error sending message")
		return "", err
	}

	return st.ID, err
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
	return *d.convertUser(u), nil
}

func (d *Discord) convertUser(u *discordgo.User) *user.User {
	img, err := d.client.UserAvatar(u.ID)
	if err != nil {
		log.Error().Err(err).Msg("error getting avatar")
	}
	return &user.User{
		ID:      u.ID,
		Name:    u.Username,
		Admin:   false,
		IconImg: img,
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

	return nil
}

func (d *Discord) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Debug().Msgf("discord message: %+v", m)
	if m.Author.ID == s.State.User.ID {
		return
	}

	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		log.Error().Err(err).Msg("error getting channel info")
	}

	isCmd, text := bot.IsCmd(d.config, m.Content)

	tStamp, _ := m.Timestamp.Parse()

	msg := msg.Message{
		ID:          m.ID,
		User:        d.convertUser(m.Author),
		Channel:     m.ChannelID,
		ChannelName: ch.Name,
		Body:        text,
		Command:     isCmd,
		Time:        tStamp,
	}

	log.Debug().Interface("m", m).Interface("msg", msg).Msg("message received")

	authorizedChannels := d.config.GetArray("channels", []string{})

	if in(ch.Name, authorizedChannels) {
		d.event(d, bot.Message, msg)
	}
}

func in(s string, lst []string) bool {
	for _, i := range lst {
		if s == i {
			return true
		}
	}
	return false
}
