package discord

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/pegnet/pegnet-node/node"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

const PegNetCommunitySlack = "550312670528798755"

type PegnetDiscordBot struct {
	token   string // Discord auth token
	session *discordgo.Session

	Node     *node.PegnetNode
	cmdRegex *regexp.Regexp
}

func NewPegnetDiscordBot(token string) (*PegnetDiscordBot, error) {
	p := new(PegnetDiscordBot)
	p.token = token

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	p.session = dg
	err = p.session.Open()
	if err != nil {
		return nil, err
	}

	p.session.AddHandler(p.messageCreate)
	p.cmdRegex, _ = regexp.Compile("!pegnet.*")


	return p, nil
}

func NewMockPegnetDiscordBot() (*PegnetDiscordBot, error) {
	p := new(PegnetDiscordBot)

	p.session.AddHandler(p.messageCreate)
	p.cmdRegex, _ = regexp.Compile("!pegnet.*")

	return p, nil
}

func (a *PegnetDiscordBot) Run(ctx context.Context) {
	for {
		select {
		case _ = <-ctx.Done():
			_ = a.Close()
			return
		default:
		}

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		text, _ := reader.ReadString('\n')
		resp := a.HandleMessage(text)
		fmt.Println(resp)
	}
}

func (a *PegnetDiscordBot) Close() error {
	return a.session.Close()
}

// Just a debug tool
func (a *PegnetDiscordBot) ListChannels() {
	for _, guild := range a.session.State.Guilds {
		channels, _ := a.session.GuildChannels(guild.ID)
		fmt.Println(guild.Name)
		for _, ch := range channels {
			fmt.Println(ch.Name, ch.ID)
		}
	}
}

func (a *PegnetDiscordBot) GetCommunitySlack() (*discordgo.Guild, error) {
	return a.session.Guild(PegNetCommunitySlack)
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (a *PegnetDiscordBot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if the message has the correct root cmd
	if !a.cmdRegex.Match([]byte(m.Content)) {
		return
	}

	data := a.HandleMessage(m.Content)
	_, _ = s.ChannelMessageSend(m.ChannelID, string(data))
}


func (a *PegnetDiscordBot) HandleMessage(input string)  string {
	out := bytes.NewBuffer([]byte{})
	a.RootCmd().SetOut(out)

	os.Args = strings.Split(input, " ")
	err := a.RootCmd().Execute()
	if err != nil {
	logrus.WithError(err).Error("root execute")
	}

	data, _ := ioutil.ReadAll(out)

	return string(data)
}

func (a *PegnetDiscordBot) s() {

}