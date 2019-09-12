package discord

import (
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// DiscordMessage is a handler for a discord message.
func (a *PegnetDiscordBot) DiscordMessage(s *discordgo.Session, m *discordgo.Message) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if the message has the correct root cmd
	if !a.cmdRegex.Match([]byte(m.Content)) {
		return
	}

	os.Args = strings.Split(m.Content, " ")
	err := a.Root(s, m).Execute()
	if err != nil {
		log.WithError(err).Error("root execute")
	}
}

func (a *PegnetDiscordBot) PrivateMessagef(session *discordgo.Session, message *discordgo.Message, format string, args ...interface{}) error {
	return a.PrivateMessage(session, message, fmt.Sprintf(format, args...))
}

func (a *PegnetDiscordBot) PrivateMessage(session *discordgo.Session, message *discordgo.Message, send string) error {
	userChannel, err := session.UserChannelCreate(message.Author.ID)
	if err != nil {
		log.WithError(err).Errorf("cannot get user channel")
		return err
	}

	_, err = session.ChannelMessage(userChannel.ID, send)
	if err != nil {
		log.WithError(err).Errorf("failed to send message")
		return err
	}

	return nil
}

func (a *PegnetDiscordBot) MessageBackf(session *discordgo.Session, message *discordgo.Message, format string, args ...interface{}) error {
	return a.MessageBack(session, message, fmt.Sprintf(format, args...))
}

func (a *PegnetDiscordBot) MessageBack(session *discordgo.Session, message *discordgo.Message, send string) error {
	_, err := session.ChannelMessage(message.ChannelID, send)
	if err != nil {
		log.WithError(err).Errorf("failed to send message")
		return err
	}
	return nil
}
