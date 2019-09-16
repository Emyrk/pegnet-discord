package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/pegnet/pegnet/opr"

	"github.com/FactomProject/factom"
	"github.com/bwmarrin/discordgo"
	"github.com/pegnet/pegnet/api"
	"github.com/pegnet/pegnet/cmd"
	"github.com/pegnet/pegnet/common"
	"github.com/spf13/cobra"
)

func (a *PegnetDiscordBot) Root(session *discordgo.Session, message *discordgo.MessageCreate) *cobra.Command {
	root := &cobra.Command{
		Use: "!pegnet <command>",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			//cmd.SetOut(out)
			//cmd.SetErr(out)
		},
		Run: func(cmd *cobra.Command, args []string) {
			var _ = cmd.Help()
		},

		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			//str := string(out.Bytes())
			//if len(str) > 0 {
			//	_ = a.CodedMessageBack(a.session, message, str)
			//}
		},
	}

	root.AddCommand(a.Performance(a.session, message))
	root.AddCommand(a.Balance(a.session, message))
	root.AddCommand(a.WhoIs(a.session, message))
	root.AddCommand(a.Winners(a.session, message))
	root.AddCommand(a.Mine(a.session, message))

	return root
}

func (a *PegnetDiscordBot) Mine(session *discordgo.Session, message *discordgo.MessageCreate) *cobra.Command {
	mine := &cobra.Command{
		Use:     "mine",
		Short:   "Begins mining PEG for the discord user that issued the command",
		Example: "!pegnet mine --miners 4 --top 3",
		Run: func(cmd *cobra.Command, args []string) {
			_ = a.CodedMessageBackf(session, message, "Sorry, I lost my pickaxe awhile back. Maybe another time.")
		},
	}

	return mine
}

func (a *PegnetDiscordBot) Supply(session *discordgo.Session, message *discordgo.MessageCreate) *cobra.Command {
	getPerformance := &cobra.Command{
		Use:       "performance total <burns|rewards>",
		Short:     "Returns the total number of FCT burns or PEG rewards ever issued",
		Example:   "!pegnet total burns\n!pegnet total rewards",
		ValidArgs: []string{"burns", "rewards"},
		Args:      cobra.OnlyValidArgs,
		Run: func(cmd *cobra.Command, args []string) {
			asset := ""
			switch args[0] {
			case "burns":
				asset = "FCT"
			case "rewards":
				asset = "PEG"
			default:
				_ = a.CodedMessageBackf(session, message, "'%s' is not a valid argument", args[0])
				return
			}

			var _ = asset

			// TODO: Finish this
		},
	}

	// RPC Wrappers
	getPerformance.Flags().Int64("start", -1, "First block in the block range requested "+
		"(negative numbers are interpreted relative to current block head)")
	getPerformance.Flags().Int64("end", -1, "Last block in the block range requested "+
		"(negative numbers are ignored)")

	return getPerformance
}

func (a *PegnetDiscordBot) Performance(session *discordgo.Session, message *discordgo.MessageCreate) *cobra.Command {
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
				_ = a.CodedMessageBack(session, message, err.Error())
				return
			}

			if resp.Err != nil {
				_ = a.CodedMessageBack(session, message, string(PrettyMarshal(resp.Err)))
				return
			}
			_ = a.CodedMessageBackf(session, message, "Hey %s, I sent you a pm with the results!", message.Author.Username)
			_ = a.CodedPrivateMessage(session, message, string(PrettyMarshal(resp.Res)))
		},
	}

	// RPC Wrappers
	getPerformance.Flags().Int64("start", -1, "First block in the block range requested "+
		"(negative numbers are interpreted relative to current block head)")
	getPerformance.Flags().Int64("end", -1, "Last block in the block range requested "+
		"(negative numbers are ignored)")

	return getPerformance
}

func (a *PegnetDiscordBot) Balance(session *discordgo.Session, message *discordgo.MessageCreate) *cobra.Command {
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
				_ = a.CodedMessageBack(session, message, "Error: invalid network string")
				return
			}
			pegAddress, err := common.ConvertFCTtoPegNetAsset(networkString, ticker, address)
			if err != nil {
				_ = a.CodedMessageBack(session, message, "Error: invalid Factoid address")
				return
			}

			bal := a.Node.PegnetGrader.Balances.GetBalance(pegAddress)

			if v, _ := cmd.Flags().GetBool("raw"); v {
				_ = a.CodedMessageBackf(session, message, "%s: %d", pegAddress, bal)
			} else {
				_ = a.CodedMessageBackf(session, message, "%s: %s %s", pegAddress, factom.FactoshiToFactoid(uint64(bal)), ticker)
			}
		},
	}

	localCmd.Flags().Bool("raw", false, "Return balances as their uint64 raw values")

	return localCmd
}

func (a *PegnetDiscordBot) Winners(session *discordgo.Session, message *discordgo.MessageCreate) *cobra.Command {
	localCmd := &cobra.Command{
		Use:     "winners <height> ",
		Short:   "Attempts to figure out all the identities/coinbase addresses for someone",
		Example: "!pegnet winners 209802",
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			targetHeight, err := strconv.Atoi(args[0])
			if err != nil {
				_ = a.CodedMessageBackf(session, message, err.Error())
				return
			}

			var block *opr.OprBlock
			if targetHeight == 0 {
				block = a.Node.PegnetGrader.GetPreviousOPRBlock(int32(math.MaxInt32 - 1))
			} else {
				block, err = a.Node.IOPRBlockStore.FetchOPRBlock(int64(targetHeight))
			}
			if err != nil {
				_ = a.CodedMessageBackf(session, message, "no block found at %d", targetHeight)
				return
			}

			if block == nil {
				_ = a.CodedMessageBackf(session, message, "There are no winners at block height %d", targetHeight)
			} else {
				str := fmt.Sprintf("Block Height %d. Total Oprs: %d", block.Dbht, block.TotalNumberRecords)
				for i, opr := range block.GradedOPRs[:a.Node.PegnetGrader.MinRecords(block.Dbht)] {
					str += fmt.Sprintf("\n  %2d %x %s", i, opr.EntryHash, opr.FactomDigitalID)
				}

				// This is for an inside joke
				if ok, _ := cmd.Flags().GetBool("extra-chrome"); ok {
					all := strings.Split(str, "")
					for i := 0; i < len(all)/40; i++ {
						index := rand.Intn(len(all))
						end := append([]string{"'"}, all[index:]...)
						all = append(all[:index], end...)
					}
					str = strings.Join(all, "")
				}
				_ = a.CodedMessageBackf(session, message, str)
			}
		},
	}

	localCmd.Flags().Bool("extra-chrome", false, "Uhhh....")
	_ = localCmd.Flags().MarkHidden("extra-chrome")

	return localCmd
}

func (a *PegnetDiscordBot) WhoIs(session *discordgo.Session, message *discordgo.MessageCreate) *cobra.Command {
	localCmd := &cobra.Command{
		Use:     "whois [factoid address|identity ...] ",
		Short:   "Attempts to figure out all the identities/coinbase addresses for someone",
		Example: "!pegnet whois FA2jK2HcLnRdS94dEcU27rF3meoJfpUcZPSinpb7AwQvPRY6RL1Q",
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			identities := make(map[string]int)
			payouts := make(map[string]int)

			next := a.Node.PegnetGrader.GetPreviousOPRBlock(math.MaxInt32)
			for {
				if next == nil {
					break
				}

				for _, opr := range next.OPRs {
					if NeedleInHackstack(args[:], opr.CoinbaseAddress) || NeedleInHackstack(args[:], opr.FactomDigitalID) {
						payouts[opr.CoinbaseAddress] += 1
						identities[opr.FactomDigitalID] += 1
					}
				}
				next = a.Node.PegnetGrader.GetPreviousOPRBlock(int32(next.Dbht))
			}

			var idStr string
			count := 0
			for id := range identities {
				idStr += fmt.Sprintf("\n\t%s", id)
				count++
				if count > 15 {
					idStr += fmt.Sprintf("\n\t... %d more not listed", len(identities)-15)
					break
				}
			}

			count = 0
			var payStr string
			for p := range payouts {
				payStr += fmt.Sprintf("\n\t%s", p)
				count++
				if count > 15 {
					payStr += fmt.Sprintf("\n\t... %d more not listed", len(identities)-15)
					break
				}
			}

			str := fmt.Sprintf("Identities%s\nPayouts%s", idStr, payStr)
			_ = a.CodedMessageBack(session, message, str)
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
