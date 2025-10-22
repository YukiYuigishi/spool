package cmd

import (
	"github.com/YukiYuigishi/spool/pkg/git"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create an empty git repository",
	Long:  "Create an empty git repository",
	Run:   initRepository,
}

func initRepository(cmd *cobra.Command, args []string) {
	git.InitGitRepository()
}

