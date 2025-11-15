package cmd

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/YukiYuigishi/spool/pkg/git"
	"github.com/spf13/cobra"
)

var catIndexCommand = &cobra.Command{
	Use:   "cat-index",
	Short: "show index file detail",
	Long:  "show index file detail",
	Run:   catIndex,
}

func catIndex(cmd *cobra.Command, args []string) {
	f, err := os.Open(".git/index")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := bufio.NewReader(f)

	var h git.IndexHeader

	if err := binary.Read(r, binary.BigEndian, &h.Signature); err != nil {
		panic(err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.Version); err != nil {
		panic(err)
	}
	if err := binary.Read(r, binary.BigEndian, &h.IndexEntrie); err != nil {
		panic(err)
	}

	fmt.Println("----index file----")
	fmt.Printf("Signature: %s\n", h.Signature)
	fmt.Printf("Version: %+v\n", h.Version)
	fmt.Printf("IndexEntire: %+v\n", h.IndexEntrie)
}
