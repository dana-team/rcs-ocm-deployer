package main

import (
	goflag "flag"
	"fmt"

	"math/rand"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"open-cluster-management.io/addon-framework/pkg/version"

	"github.com/dana-team/rcs-ocm-deployer/addons/score/scorehub"
	"github.com/dana-team/rcs-ocm-deployer/addons/score/scorespoke"
)

func main() {
	rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	zapLog, err := zap.NewDevelopment()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to start up logger %v\n", err)
		os.Exit(1)
	}

	logger := zapr.NewLogger(zapLog)

	command := newCommand(logger)
	if err := command.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// newCommand creates a new Cobra command for the score-addon CLI.
// It initializes the root command with the appropriate metadata and subcommands.
func newCommand(logger logr.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "score-addon",
		Short: "score-addon",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			os.Exit(1)
		},
	}

	if v := version.Get().String(); len(v) == 0 {
		cmd.Version = "<unknown>"
	} else {
		cmd.Version = v
	}

	cmd.AddCommand(scorehub.NewManagerCommand("rcs-score-addon-controller", logger.WithName("rcs-score-addon-controller")))
	cmd.AddCommand(scorespoke.NewAgentCommand("rcs-score-addon-agent", scorehub.AddonName, logger.WithName("rcs-score-addon-controller")))

	return cmd
}
