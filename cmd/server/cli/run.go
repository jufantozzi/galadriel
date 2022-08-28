package cli

import (
	"github.com/HewlettPackard/galadriel/cmd/server/api"
	"github.com/HewlettPackard/galadriel/pkg/server/config"
	"github.com/spf13/cobra"
)

const defaultConfigPath = "conf/server/server.conf"

var configPath string
var runServerFn = ServerCLI.runServerAPI
var runAPI = api.Run
var loadConfigFromDisk = config.LoadFromDisk

func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Runs the Galadriel server",
		Long:  "Run this command to start the Galadriel server",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}

			runServerFn(configPath)
			return nil
		},
	}
}

func (c *serverCLI) runServerAPI(configPath string) {
	c.logger.Info("Starting Galadriel Server")

	// TODO: pass config variables to runServerFn()
	_, err := loadConfigFromDisk(configPath)
	if err != nil {
		c.logger.Error("Error loading config:", err)
		return
	}

	runAPI()
}

func init() {
	runCmd := NewRunCmd()
	runCmd.Flags().StringVarP(&configPath, "config", "c", defaultConfigPath, "config file path")

	RootCmd.AddCommand(runCmd)
}
