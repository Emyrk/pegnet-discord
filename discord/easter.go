package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// EasterEggHandling... I had too
func (a *PegnetDiscordBot) EasterEggHandling(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	if a.ReturnWave(s, m) {
		return true
	}

	return false
}

func (a *PegnetDiscordBot) ReturnWave(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	if strings.Contains(m.Content, "👋") {
		for _, mention := range m.Mentions {
			fmt.Println(mention)
			if mention.Username == "Mr Pegnet" {
				_ = a.MessageBackf(a.session, m, ":wave: %s", m.Author.Mention())
				return true
			}
		}
	}
	return false
}
