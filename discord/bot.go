package discord

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/pegnet/pegnet-node/node"
	"github.com/pegnet/pegnet/api"
	"github.com/pegnet/pegnet/common"
	config "github.com/zpatrick/go-config"
)

const PegNetCommunitySlack = "550312670528798755"
const TestZoneChannel = "621819950851555358"
const BotCommunicationChannel = "622075411123273751"

var BlackListedChannels = map[string]bool{
	"550312670528798761": true,
}

type PegnetDiscordBot struct {
	token   string // Discord auth token
	session *discordgo.Session

	config   *config.Config
	Node     *node.PegnetNode
	API      *api.APIServer
	cmdRegex *regexp.Regexp

	// TODO: This is hacky
	sync.Mutex
	returnChannel string
}

func NewPegnetDiscordBot(token string, config *config.Config) (*PegnetDiscordBot, error) {
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

	p.session.AddHandler(p.DiscordMessage)
	p.cmdRegex, _ = regexp.Compile("!pegnet.*")
	p.config = config

	common.GlobalExitHandler.AddExit(func() error {
		msg := fmt.Sprintf(":x:  I'm going offline. Sorry for the inconvience, but I'm not being hosted on the cloud yet :cry: ")
		_, err := p.session.ChannelMessageSend(BotCommunicationChannel, msg)
		return err
	})

	msg := ":white_check_mark:  Hey I'm back online!"
	_, _ = p.session.ChannelMessageSend(BotCommunicationChannel, msg)

	return p, nil
}

func NewMockPegnetDiscordBot(config *config.Config) (*PegnetDiscordBot, error) {
	p := new(PegnetDiscordBot)

	p.cmdRegex, _ = regexp.Compile("!pegnet.*")
	p.config = config

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
		// resp := a.HandleMessage(text)
		fmt.Println("Sorry...", text)
		// fmt.Println(resp)
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
