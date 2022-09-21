package cli

import (
	"context"
	"fmt"
	"github.com/HewlettPackard/galadriel/pkg/common/telemetry"
	"github.com/HewlettPackard/galadriel/pkg/common/util"
	"github.com/HewlettPackard/galadriel/pkg/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

const defaultConfigPath = "conf/server/server.conf"

var configPath string

func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Runs the Galadriel server",
		Long:  "Run this command to start the Galadriel server",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := LoadConfig(cmd)
			if err != nil {
				return err
			}

			s := server.New(config)

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			ctx = util.WithFields(ctx, logrus.Fields{
				telemetry.SubsystemName: telemetry.GaladrielServer,
				"DataDir":               config.DataDir,
			})
			defer stop()

			err = s.Run(ctx)
			if err != nil {
				return err
			}

			config.Log.Info("Server stopped gracefully")
			return nil
		},
	}
}

func LoadConfig(cmd *cobra.Command) (*server.Config, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, fmt.Errorf("cannot read flag config: %w", err)
	}

	if configPath == "" {
		configPath = defaultConfigPath
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open configuration file: %w", err)
	}
	defer configFile.Close()

	c, err := ParseConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	sc, err := NewServerConfig(c)
	if err != nil {
		return nil, fmt.Errorf("failed to build server configuration: %w", err)
	}

	return sc, nil
}

func init() {
	runCmd := NewRunCmd()
	runCmd.Flags().StringVarP(&configPath, "config", "c", defaultConfigPath, "config file path")

	RootCmd.AddCommand(runCmd)
}
