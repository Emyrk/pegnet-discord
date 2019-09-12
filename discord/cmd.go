package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"

	"github.com/pegnet/pegnet/api"

	"github.com/FactomProject/factom"

	"github.com/pegnet/pegnet/cmd"
	"github.com/pegnet/pegnet/common"
	"github.com/spf13/cobra"
)

func (a *PegnetDiscordBot) RootCmd() *cobra.Command {
	root := &cobra.Command{
		Use: "!pegnet <command>",
		Run: func(cmd *cobra.Command, args []string) {
			var _ = cmd.Help()
		},
	}

	root.AddCommand(a.Balance())
	root.AddCommand(a.Performance())
	return root
}

func (a *PegnetDiscordBot) Performance() *cobra.Command {
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
				HandleError(cmd, err)
				return
			}

			if resp.Err != nil {
				HandleErrorStr(cmd, string(PrettyMarshal(resp.Err)))
				return
			}

			Printf(cmd, string(PrettyMarshal(resp.Res)))
		},
	}

	// RPC Wrappers
	getPerformance.Flags().Int64("start", -1, "First block in the block range requested "+
		"(negative numbers are interpreted relative to current block head)")
	getPerformance.Flags().Int64("end", -1, "Last block in the block range requested "+
		"(negative numbers are ignored)")

	return getPerformance
}

func (a *PegnetDiscordBot) Balance() *cobra.Command {
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
				HandleError(cmd, fmt.Errorf("Error: invalid network string"))
				return
			}
			pegAddress, err := common.ConvertFCTtoPegNetAsset(networkString, ticker, address)
			if err != nil {
				HandleError(cmd, fmt.Errorf("Error: invalid Factoid address"))
				return
			}

			bal := a.Node.PegnetGrader.Balances.GetBalance(pegAddress)

			if v, _ := cmd.Flags().GetBool("raw"); v {
				Printf(cmd, "%s: %d", pegAddress, bal)
			} else {
				Printf(cmd, "%s: %s", pegAddress, factom.FactoshiToFactoid(uint64(bal)))
			}
		},
	}

	localCmd.Flags().Bool("raw", false, "Return balances as their uint64 raw values")

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

func Printf(cmd *cobra.Command, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(cmd.OutOrStderr(), format, args...)
}

func HandleErrorStr(cmd *cobra.Command, err string) {
	HandleError(cmd, fmt.Errorf(err))
}

func HandleError(cmd *cobra.Command, err error) {
	_, _ = fmt.Fprintf(cmd.OutOrStderr(), err.Error())
}
