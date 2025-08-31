package discord

import (
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const CommandName = "Backup"

type Bot struct {
	appID   string
	session *discordgo.Session
	logger  *zap.Logger

	Interactions []*Interaction
}

type InteractionCallback func(filename string, url string, messageTime time.Time) (string, error)

type Interaction struct {
	ID                string
	Name              string
	AuthorizedUserIDs []string
	Callback          InteractionCallback
}

func NewBot(logger *zap.Logger, appID string, token string) (*Bot, error) {
	bot := Bot{
		appID:  appID,
		logger: logger,
	}

	// Create a new Discord session
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		logger.Error("failed to create Discord session", zap.Error(err))
		return nil, err
	}

	// Handler to know when the bot is registered and ready on discord
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		logger.Info("bot is up")
	})

	bot.session = s

	return &bot, nil
}

func (b *Bot) AddInteraction(name string, authorizedUserIDs []string, callback InteractionCallback) error {
	// Add the backupInteraction handler
	b.session.AddHandler(b.botInteraction)

	c, err := b.session.ApplicationCommandCreate(b.appID, "", &discordgo.ApplicationCommand{
		Name: name,
		Type: discordgo.MessageApplicationCommand,
	})

	if err != nil {
		b.logger.Error("creating interaction", zap.String("name", name), zap.Error(err))
		return err
	}

	b.Interactions = append(b.Interactions, &Interaction{
		ID:                c.ID,
		Name:              name,
		AuthorizedUserIDs: authorizedUserIDs,
		Callback:          callback,
	})

	b.logger.Info("created interation", zap.String("name", name))
	return nil
}

func (b *Bot) Start() error {
	// Open a websocket connection to Discord and begin listening
	err := b.session.Open()
	if err != nil {
		b.logger.Error("opening connection", zap.Error(err))
		return err
	}

	return nil
}

func (b *Bot) Shutdown() error {
	defer b.session.Close()

	for _, i := range b.Interactions {
		if err := b.session.ApplicationCommandDelete(b.appID, "", i.ID); err != nil {
			b.logger.Error("deleting interaction", zap.Error(err))
			return err
		}
		b.logger.Info("deleted interaction", zap.String("name", i.Name))
	}

	b.logger.Info("bot is down")

	return nil
}

func (b *Bot) botInteraction(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		b.logger.Error("can't send interaction response", zap.Error(err))
		return
	}

	var statuses string

	for _, i := range b.Interactions {
		if interaction.ApplicationCommandData().Name == i.Name {
			// Check if an authorized user is interacting
			var user *discordgo.User
			if interaction.User != nil {
				user = interaction.User
			} else if interaction.Member.User != nil {
				user = interaction.Member.User
			}

			if !slices.Contains(i.AuthorizedUserIDs, user.ID) {
				b.logger.Info("interaction detected but is not authorized for that user", zap.String("name", user.Username))
				return
			}

			// Loop through the attachments in the message
			message := interaction.ApplicationCommandData().Resolved.Messages[interaction.ApplicationCommandData().TargetID]
			messageTime, err := parseSnowflakeToTime(message.ID)

			if err != nil {
				b.logger.Error("failed to parse message ID to time", zap.Error(err))
			}

			if len(message.Attachments) > 0 {
				for _, a := range message.Attachments {
					// Check if the attachment is an image
					if strings.HasPrefix(a.ContentType, "image/") || strings.HasPrefix(a.ContentType, "video/") {
						status, err := i.Callback(a.Filename, a.URL, messageTime)
						if err != nil {
							b.logger.Error("failed callback", zap.String("image", a.Filename), zap.Error(err))
						} else {
							b.logger.Info("callback successfully", zap.String("image", a.Filename))
						}
						statuses += status
					}
				}
			} else {
				b.logger.Info("no attachements detected")
				statuses = "no attachments detected"
			}
		}
	}

	_, err = session.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
		Content: statuses,
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	if err != nil {
		b.logger.Error("can't update interaction message", zap.Error(err))
	}
}
