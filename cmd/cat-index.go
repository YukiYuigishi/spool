package cmd

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/YukiYuigishi/spool/pkg/git"
	"github.com/spf13/cobra"
)

func NewCatIndexCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "cat-index",
		Short: "cat git index file",
		Long:  "cat git index file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCatIndexCmd(cmd, args)
		},
	}
	return cmd
}

func runCatIndexCmd(_ *cobra.Command, _ []string) error {
	f, err := os.Open(".git/index")
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	var h git.IndexHeader

	if err := binary.Read(r, binary.BigEndian, &h.Signature); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &h.Version); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &h.IndexEntries); err != nil {
		return err
	}

	fmt.Println("----index file----")
	fmt.Printf("Signature: %s\n", h.Signature)
	fmt.Printf("Version: %+v\n", h.Version)
	fmt.Printf("IndexEntire: %+v\n", h.IndexEntries)
	return nil
}
