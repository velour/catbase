package discord

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

	// store IDs -> nick and vice versa for quick conversion
	uidCache map[string]string

	registeredCmds []*discordgo.ApplicationCommand
	cmdHandlers    map[string]CmdHandler

	guildID string
}

func New(config *config.Config) *Discord {
	client, err := discordgo.New("Bot " + config.Get("DISCORDBOTTOKEN", ""))
	if err != nil {
		log.Fatal().Err(err).Msg("Could not connect to Discord")
	}
	guildID := config.Get("discord.guildid", "")
	if guildID == "" {
		log.Fatal().Msgf("You must set either DISCORD_GUILDID env or discord.guildid db config")
	}
	d := &Discord{
		config:         config,
		client:         client,
		uidCache:       map[string]string{},
		registeredCmds: []*discordgo.ApplicationCommand{},
		cmdHandlers:    map[string]CmdHandler{},
		guildID:        guildID,
	}
	d.client.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := d.cmdHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
	return d
}
func (d *Discord) GetRouter() (http.Handler, string) {
	return nil, ""
}

func (d *Discord) RegisterEvent(callback bot.Callback) {
	d.event = callback
}

func (d Discord) Send(kind bot.Kind, args ...any) (string, error) {

	switch kind {
	case bot.Ephemeral:
		return d.sendMessage(args[0].(string), args[2].(string), false, args...)
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
			msg = fmt.Sprintf("> %v\n%s", original, msg)
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

func (d *Discord) sendMessage(channel, message string, meMessage bool, args ...any) (string, error) {
	if meMessage && !strings.HasPrefix(message, "_") && !strings.HasSuffix(message, "_") {
		message = "_" + message + "_"
	}

	embeds := []*discordgo.MessageEmbed{}
	files := []*discordgo.File{}

	for _, arg := range args {
		switch a := arg.(type) {
		case bot.EmbedAuthor:
			embed := &discordgo.MessageEmbed{}
			embed.Author = &discordgo.MessageEmbedAuthor{
				Name:    a.Who,
				IconURL: a.IconURL,
			}
			embeds = append(embeds, embed)
		case bot.ImageAttachment:
			embed := &discordgo.MessageEmbed{}
			embed.Description = a.AltTxt
			embed.Image = &discordgo.MessageEmbedImage{
				URL:    a.URL,
				Width:  a.Width,
				Height: a.Height,
			}
			embeds = append(embeds, embed)
		case bot.File:
			files = append(files, &discordgo.File{
				Name:        a.FileName(),
				ContentType: a.ContentType(),
				Reader:      bytes.NewBuffer(a.Data),
			})
		}
	}

	data := &discordgo.MessageSend{
		Content: message,
		Embeds:  embeds,
		Files:   files,
	}

	log.Debug().
		Interface("data", data).
		Msg("sending message")

	st, err := d.client.ChannelMessageSendComplex(channel, data)

	//st, err := d.client.ChannelMessageSend(channel, message)
	if err != nil {
		log.Error().
			Interface("data", data).
			Str("channel", channel).
			Err(err).
			Msg("Error sending message")
		return "", err
	}

	return st.ID, err
}

func (d *Discord) GetEmojiList(force bool) map[string]string {
	if d.emojiCache != nil && !force {
		return d.emojiCache
	}
	e, err := d.client.GuildEmojis(d.guildID)
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

func (d *Discord) GetEmojySnowflake(name string) string {
	list := d.GetEmojiList(true)
	for k, v := range list {
		if name == v {
			return k
		}
	}
	return name
}

// Who gets the users in the guild
// Note that the channel ID does not matter in this particular case
func (d *Discord) Who(id string) []string {
	users := []string{}
	for name, _ := range d.uidCache {
		users = append(users, name)
	}
	return users
}

func (d *Discord) Profile(id string) (user.User, error) {
	if _, err := strconv.Atoi(id); err != nil {
		if newID, ok := d.uidCache[id]; ok {
			id = newID
		} else {
			return user.User{}, fmt.Errorf("invalid ID snowflake: %s", id)
		}
	}
	u, err := d.client.User(id)
	if err != nil {
		log.Error().Err(err).Msg("Error getting user")
		return user.User{}, err
	}
	user := d.convertUser(u)
	d.uidCache[user.Name] = user.ID
	return *user, nil
}

func (d *Discord) convertUser(u *discordgo.User) *user.User {
	log.Debug().
		Str("u.ID", u.ID).
		Msg("")
	img, err := d.client.UserAvatar(u.ID)
	if err != nil {
		log.Error().Err(err).Msg("error getting avatar")
	}
	nick := u.Username

	mem, err := d.client.GuildMember(d.guildID, u.ID)
	if err != nil {
		log.Error().Err(err).Msg("could not get guild member")
	} else if mem.Nick != "" {
		nick = mem.Nick
	}

	return &user.User{
		ID:      u.ID,
		Name:    nick,
		Admin:   false,
		Icon:    d.client.State.User.AvatarURL("64"),
		IconImg: img,
	}
}

func (d *Discord) Serve() error {
	log.Debug().Msg("starting discord serve function")

	d.client.Identify.Intents = discordgo.MakeIntent(
		discordgo.IntentsGuilds |
			discordgo.IntentsGuildMessages |
			discordgo.IntentsDirectMessages |
			discordgo.IntentsGuildEmojis |
			discordgo.IntentsGuildMessageReactions)

	err := d.client.Open()
	if err != nil {
		log.Fatal().Err(err).Msg("error opening client")
		return err
	}

	log.Debug().Msg("discord connection open")

	d.client.AddHandler(d.messageCreate)
	d.client.AddHandler(d.messageReactAdd)

	return nil
}

func (d *Discord) messageReactAdd(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	log.Debug().Msg("messageReactAdd")
	author, _ := d.Profile(m.UserID)
	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		log.Error().Err(err).Msg("error getting channel info")
	}
	msg := msg.Message{
		ID:          m.MessageID,
		Kind:        bot.Reaction,
		User:        &author,
		Channel:     m.ChannelID,
		ChannelName: ch.Name,
		Body:        m.Emoji.Name,
		Time:        time.Now(),
		AdditionalData: map[string]string{
			"messageFormat": m.Emoji.MessageFormat(),
		},
	}
	d.event(d, bot.Reaction, msg)
}

func (d *Discord) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// if we haven't seen this user, we need to resolve their nick before continuing
	author, _ := d.Profile(m.Author.ID)

	if m.Author.ID == s.State.User.ID {
		return
	}

	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		log.Error().Err(err).Msg("error getting channel info")
	}

	isCmd, text := bot.IsCmd(d.config, m.Content)

	tStamp := m.Timestamp

	msg := msg.Message{
		ID:          m.ID,
		Kind:        bot.Message,
		User:        &author,
		Channel:     m.ChannelID,
		ChannelName: ch.Name,
		Body:        text,
		Command:     isCmd,
		Time:        tStamp,
	}

	d.event(d, bot.Message, msg)
}

func (d *Discord) Emojy(name string) string {
	e := d.config.GetMap("discord.emojy", map[string]string{})
	if emojy, ok := e[name]; ok {
		return emojy
	}
	return name
}

func (d *Discord) UploadEmojy(emojy, path string) error {
	defaultRoles := d.config.GetArray("discord.emojyRoles", []string{})
	_, err := d.client.GuildEmojiCreate(d.guildID, &discordgo.EmojiParams{
		emojy, path, defaultRoles,
	})
	if err != nil {
		return err
	}
	return nil
}

func (d *Discord) DeleteEmojy(emojy string) error {
	emojyID := d.GetEmojySnowflake(emojy)
	return d.client.GuildEmojiDelete(d.guildID, emojyID)
}

func (d *Discord) URLFormat(title, url string) string {
	return fmt.Sprintf("%s (%s)", title, url)
}

// GetChannelName returns the channel ID for a human-friendly name (if possible)
func (d *Discord) GetChannelID(name string) string {
	chs, err := d.client.GuildChannels(d.guildID)
	if err != nil {
		return name
	}
	for _, ch := range chs {
		if ch.Name == name {
			return ch.ID
		}
	}
	return name
}

// GetChannelName returns the human-friendly name for an ID (if possible)
func (d *Discord) GetChannelName(id string) string {
	ch, err := d.client.Channel(id)
	if err != nil {
		return id
	}
	return ch.Name
}

func (d *Discord) GetRoles() ([]bot.Role, error) {
	ret := []bot.Role{}

	roles, err := d.client.GuildRoles(d.guildID)
	if err != nil {
		return nil, err
	}

	for _, r := range roles {
		ret = append(ret, bot.Role{
			ID:   r.ID,
			Name: r.Name,
		})
	}

	return ret, nil
}

func (d *Discord) SetRole(userID, roleID string) error {
	member, err := d.client.GuildMember(d.guildID, userID)
	if err != nil {
		return err
	}
	for _, r := range member.Roles {
		if r == roleID {
			return d.client.GuildMemberRoleRemove(d.guildID, userID, roleID)
		}
	}
	return d.client.GuildMemberRoleAdd(d.guildID, userID, roleID)
}

type CmdHandler func(s *discordgo.Session, i *discordgo.InteractionCreate)

func (d *Discord) RegisterSlashCmd(c discordgo.ApplicationCommand, handler CmdHandler) error {
	cmd, err := d.client.ApplicationCommandCreate(d.client.State.User.ID, d.guildID, &c)
	d.cmdHandlers[c.Name] = handler
	if err != nil {
		return err
	}
	d.registeredCmds = append(d.registeredCmds, cmd)
	return nil
}

func (d *Discord) Shutdown() {
	log.Debug().Msgf("Shutting down and deleting %d slash commands", len(d.registeredCmds))
	for _, c := range d.registeredCmds {
		if err := d.client.ApplicationCommandDelete(d.client.State.User.ID, d.guildID, c.ID); err != nil {
			log.Error().Err(err).Msgf("could not delete command %s", c.Name)
		}
	}
}

func (d *Discord) Nick(nick string) error {
	return d.client.GuildMemberNickname(d.guildID, "@me", nick)
}

func (d *Discord) Topic(channelID string) (string, error) {
	channel, err := d.client.Channel(channelID)
	if err != nil {
		return "", err
	}
	return channel.Topic, nil
}

func (d *Discord) SetTopic(channelID, topic string) error {
	ce := &discordgo.ChannelEdit{
		Topic: topic,
	}
	_, err := d.client.ChannelEditComplex(channelID, ce)
	return err
}

type ThreadStart struct {
	Name                string           `json:"name"`
	AutoArchiveDuration int              `json:"auto_archive_duration,omitempty"`
	RateLimitPerUser    int              `json:"rate_limit_per_user,omitempty"`
	AppliedTags         []string         `json:"applied_tags,omitempty"`
	Message             ForumMessageData `json:"message"`
}

type ForumMessageData struct {
	Content string `json:"content"`
}

func (d *Discord) CreateRoom(name, message, parent string, duration int) (string, error) {
	data := &ThreadStart{
		Name:                name,
		AutoArchiveDuration: duration,
		Message:             ForumMessageData{message},
	}
	ch := &discordgo.Channel{}
	endpoint := discordgo.EndpointChannelThreads(parent)
	body, err := d.client.RequestWithBucketID("POST", endpoint, data, endpoint)
	if err != nil {
		return "", err
	}

	if err = discordgo.Unmarshal(body, &ch); err != nil {
		return "", err
	}
	return ch.ID, nil
}
