package discord

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// DiscordMessage is a handler for a discord message.
func (a *PegnetDiscordBot) DiscordMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	// Reject Mr or Junior
	if m.Author.ID == s.State.User.ID || m.Author.ID == "622875359813042190" || m.Author.ID == "621815369962881024" {
		return
	}

	if a.EasterEggHandling(s, m) {
		return // Hehe
	}

	// Check if the message has the correct root cmd
	if !a.cmdRegex.Match([]byte(m.Content)) {
		return
	}

	os.Args = strings.Split(m.Content, " ")

	root := a.Root(s, m)
	out := bytes.NewBuffer([]byte{})
	root.SetOut(out)
	root.SetErr(out)

	err := root.Execute()
	if err != nil {
		log.WithError(err).Error("root execute")
	}

	str := string(out.Bytes())
	if len(str) > 0 {
		_ = a.CodedMessageBack(a.session, m, str)
	}
}

func (a *PegnetDiscordBot) CodedPrivateMessagef(session *discordgo.Session, message *discordgo.MessageCreate, format string, args ...interface{}) error {
	return a.CodedPrivateMessage(session, message, fmt.Sprintf(format, args...))
}

func (a *PegnetDiscordBot) CodedPrivateMessage(session *discordgo.Session, message *discordgo.MessageCreate, send string) error {
	userChannel, err := session.UserChannelCreate(message.Author.ID)
	if err != nil {
		log.WithError(err).Errorf("cannot get user channel")
		return err
	}

	_, err = session.ChannelMessageSend(userChannel.ID, fmt.Sprintf("```%s```", send))
	if err != nil {
		log.WithError(err).Errorf("failed to send message")
		return err
	}

	return nil
}

func (a *PegnetDiscordBot) MessageBackf(session *discordgo.Session, message *discordgo.MessageCreate, format string, args ...interface{}) error {
	return a.MessageBack(session, message, fmt.Sprintf(format, args...))
}

func (a *PegnetDiscordBot) MessageBack(session *discordgo.Session, message *discordgo.MessageCreate, send string) error {
	_, err := session.ChannelMessageSend(message.ChannelID, send)
	if err != nil {
		log.WithError(err).Errorf("failed to send message")
		return err
	}
	return nil
}

func (a *PegnetDiscordBot) CodedMessageBackf(session *discordgo.Session, message *discordgo.MessageCreate, format string, args ...interface{}) error {
	return a.CodedMessageBack(session, message, fmt.Sprintf(format, args...))
}

func (a *PegnetDiscordBot) CodedMessageBack(session *discordgo.Session, message *discordgo.MessageCreate, send string) error {
	return a.MessageBack(session, message, fmt.Sprintf("```\n%s```", send))
}
