package cli

import (
	"context"
	"fmt"
	"github.com/HewlettPackard/galadriel/pkg/server"
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
			configPath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}

			configFile, err := os.Open(configPath)
			if err != nil {
				return fmt.Errorf("unable to open configuration file: %v", err)
			}
			defer configFile.Close()

			c, err := ParseConfig(configFile)
			if err != nil {
				return err
			}

			sc, err := NewServerConfig(c)
			if err != nil {
				return err
			}

			s := server.New(sc)

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			err = s.Run(ctx)
			if err != nil {
				return err
			}

			sc.Log.Info("Server stopped gracefully")
			return nil
		},
	}
}

func init() {
	runCmd := NewRunCmd()
	runCmd.Flags().StringVarP(&configPath, "config", "c", defaultConfigPath, "config file path")

	RootCmd.AddCommand(runCmd)
}
