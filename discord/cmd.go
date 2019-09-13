package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http/httptest"

	"github.com/FactomProject/factom"
	"github.com/bwmarrin/discordgo"
	"github.com/pegnet/pegnet/api"
	"github.com/pegnet/pegnet/cmd"
	"github.com/pegnet/pegnet/common"
	"github.com/spf13/cobra"
)

func (a *PegnetDiscordBot) Root(session *discordgo.Session, message *discordgo.Message) *cobra.Command {
	root := &cobra.Command{
		Use: "!pegnet <command>",
		PreRun: func(cmd *cobra.Command, args []string) {
			out := bytes.NewBuffer([]byte{})
			cmd.SetOutput(out)
			_ = a.MessageBack(session, message, string(out.Bytes()))
		},
		Run: func(cmd *cobra.Command, args []string) {
			var _ = cmd.Help()
		},
	}

	root.AddCommand(a.Performance(session, message))
	root.AddCommand(a.Balance(session, message))
	// root.AddCommand(a.WhoIs(session, message))

	return root
}

func (a *PegnetDiscordBot) Performance(session *discordgo.Session, message *discordgo.Message) *cobra.Command {
	getPerformance := &cobra.Command{
		Use:   "performance <miner identifier> [--start START_BLOCK] [--end END_BLOCK]",
		Short: "Returns the performance of the miner at the specified identifier.",
		Long: "Every block, miners submissions are first ranked according to hash-power/difficulty computed, then by " +
			"the quality of their pricing estimates.\nThis function returns statistics to evaluate where a given miner " +
			"stands in the rankings for both categories over a specific range of blocks.",
		Example: "!pegnet performance prototypeminer001 --start=-144",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]

			blockRangeStart, _ := cmd.Flags().GetInt64("start")
			blockRangeEnd, _ := cmd.Flags().GetInt64("end")

			blockRange := api.BlockRange{Start: &blockRangeStart}
			if blockRangeEnd > 0 {
				blockRange.End = &blockRangeEnd
			}

			req := api.PostRequest{
				Method: "performance",
				Params: api.PerformanceParameters{
					BlockRange: blockRange,
					DigitalID:  id,
				},
			}

			resp, err := a.HandleAPIRequest(&req)
			if err != nil {
				_ = a.MessageBack(session, message, err.Error())
				return
			}

			if resp.Err != nil {
				_ = a.MessageBack(session, message, string(PrettyMarshal(resp.Err)))
				return
			}

			_ = a.PrivateMessage(session, message, string(PrettyMarshal(resp.Res)))
		},
	}

	// RPC Wrappers
	getPerformance.Flags().Int64("start", -1, "First block in the block range requested "+
		"(negative numbers are interpreted relative to current block head)")
	getPerformance.Flags().Int64("end", -1, "Last block in the block range requested "+
		"(negative numbers are ignored)")

	return getPerformance
}

func (a *PegnetDiscordBot) Balance(session *discordgo.Session, message *discordgo.Message) *cobra.Command {
	localCmd := &cobra.Command{
		Use:     "balance <type> <factoid address>",
		Short:   "Returns the balance for the given asset type and Factoid address",
		Example: "!pegnet balance PEG FA2jK2HcLnRdS94dEcU27rF3meoJfpUcZPSinpb7AwQvPRY6RL1Q",
		Args:    cmd.CombineCobraArgs(cmd.CustomArgOrderValidationBuilder(true, cmd.ArgValidatorAsset, cmd.ArgValidatorFCTAddress)),
		Run: func(cmd *cobra.Command, args []string) {
			ticker := args[0]
			address := args[1]

			networkString, err := common.LoadConfigNetwork(a.config)
			if err != nil {
				_ = a.MessageBack(session, message, "Error: invalid network string")
				return
			}
			pegAddress, err := common.ConvertFCTtoPegNetAsset(networkString, ticker, address)
			if err != nil {
				_ = a.MessageBack(session, message, "Error: invalid Factoid address")
				return
			}

			bal := a.Node.PegnetGrader.Balances.GetBalance(pegAddress)

			if v, _ := cmd.Flags().GetBool("raw"); v {
				_ = a.MessageBackf(session, message, "%s: %d", pegAddress, bal)
			} else {
				_ = a.MessageBackf(session, message, "%s: %s %s", pegAddress, factom.FactoshiToFactoid(uint64(bal)), ticker)
			}
		},
	}

	localCmd.Flags().Bool("raw", false, "Return balances as their uint64 raw values")

	return localCmd
}

func (a *PegnetDiscordBot) WhoIs(session *discordgo.Session, message *discordgo.Message) *cobra.Command {
	localCmd := &cobra.Command{
		Use:     "whois [factoid address|identity ...] ",
		Short:   "Attempts to figure out all the identities/coinbase addresses for someone",
		Example: "!pegnet whois FA2jK2HcLnRdS94dEcU27rF3meoJfpUcZPSinpb7AwQvPRY6RL1Q",
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var identities []string
			var payouts []string

			next := a.Node.PegnetGrader.GetPreviousOPRBlock(math.MaxInt32)
			for {
				if next == nil {
					break
				}

				for _, opr := range next.OPRs {
					if NeedleInHackstack(args[1:], opr.CoinbaseAddress) {
						payouts = append(payouts, opr.CoinbaseAddress)
					}

					if NeedleInHackstack(args[1:], opr.FactomDigitalID) {
						identities = append(identities, opr.FactomDigitalID)
					}
				}
				next = a.Node.PegnetGrader.GetPreviousOPRBlock(int32(next.Dbht))
			}

			var idStr string
			for _, id := range identities {
				idStr += fmt.Sprintf("%s\t%s", "\n", id)
			}

			var payStr string
			for _, p := range payouts {
				idStr += fmt.Sprintf("%s\t%s", "\n", p)
			}

			_ = a.MessageBackf(session, message, "Identities%s\nPayouts%s", idStr, payStr)
		},
	}

	return localCmd
}

func (a *PegnetDiscordBot) HandleAPIRequest(req *api.PostRequest) (*api.PostResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq := httptest.NewRequest("POST", "/performance", bytes.NewBuffer(data))
	resp := httptest.NewRecorder()
	a.API.ServeHTTP(resp, httpReq)

	data, _ = ioutil.ReadAll(resp.Body)

	var apiResp api.PostResponse
	err = json.Unmarshal(data, &apiResp)
	return &apiResp, err
}

func PrettyMarshal(v interface{}) []byte {
	d, _ := json.Marshal(v)
	var buf bytes.Buffer
	_ = json.Indent(&buf, d, "", "  ")
	return buf.Bytes()
}

func NeedleInHackstack(haystack []string, needle string) bool {
	for _, a := range haystack {
		if needle == a {
			return true
		}
	}
	return false
}
