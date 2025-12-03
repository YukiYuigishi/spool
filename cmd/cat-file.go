package cmd

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/YukiYuigishi/spool/pkg/git"
	"github.com/spf13/cobra"
)

type CatFileOptions struct {
	Type   bool
	Size   bool
	Pretty bool
	Exist  bool
}

func NewCatFileCmd() *cobra.Command {
	opts := &CatFileOptions{}

	cmd := &cobra.Command{
		Use:   "cat-file",
		Short: "cat git object file",
		Long:  "cat git object file",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			count := 0
			if opts.Type {
				count++
			}
			if opts.Size {
				count++
			}
			if opts.Pretty {
				count++
			}
			if opts.Exist {
				count++
			}

			if count == 0 {
				return fmt.Errorf("one of -e, -p, -t, -s must be specified")
			}
			if count > 1 {
				return fmt.Errorf("only of -e, -p, -t, -s can be specified")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCatFileCmd(cmd, opts, args)
		},
	}
	// flagをかく
	flags := cmd.Flags()
	flags.BoolVar(&opts.Exist, "e", false, "exist")
	flags.BoolVarP(&opts.Pretty, "pretty", "p", false, "pretty print")
	flags.BoolVar(&opts.Type, "t", false, "type")
	flags.BoolVar(&opts.Size, "s", false, "size")
	return cmd
}

func runCatFileCmd(_ *cobra.Command, opts *CatFileOptions, args []string) error {
	if len(args) < 1 {
		return errors.New("require <object>")
	}
	gitroot, err := ResolveForGitRoot()
	if err != nil {
		return err
	}
	objectsdir := filepath.Join(gitroot, ".git", "objects")
	hash, err := git.ExpandHashPrefix(objectsdir, args[0])
	if err != nil {
		return err
	}
	switch {
	case opts.Pretty:
		return printObject(objectsdir, hash)
	case opts.Type:
		return printObjectType(objectsdir, hash)
	case opts.Size:
		return printObjectSize(objectsdir, hash)
	case opts.Exist:
		return checkObjectExists(objectsdir, hash)
	}
	return nil
}

func printObject(objectsDir string, hash [sha1.Size]byte) error {
	obj, body, err := git.ReadObject(objectsDir, hash[:])
	if err != nil {
		return err
	}

	switch obj.Type {
	case git.ObjectTypeBlob, git.ObjectTypeCommit, git.ObjectTypeTag:
		fmt.Print(string(body))
	case git.ObjectTypeTree:
		entries, err := parseTreeEntries(body)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			fmt.Printf("%s %s %s\t%s\n", entry.Mode, treeEntryType(entry.Mode), hex.EncodeToString(entry.Hash[:]), entry.Name)
		}
	default:
		return fmt.Errorf("pretty print not supported for %s", obj.Type)
	}
	return nil
}

func printObjectType(objectsDir string, hash [sha1.Size]byte) error {
	obj, _, err := git.ReadObject(objectsDir, hash[:])
	if err != nil {
		return err
	}
	fmt.Println(obj.Type)
	return nil
}

func printObjectSize(objectsDir string, hash [sha1.Size]byte) error {
	obj, _, err := git.ReadObject(objectsDir, hash[:])
	if err != nil {
		return err
	}
	fmt.Println(obj.Size)
	return nil
}

func checkObjectExists(objectsDir string, hash [sha1.Size]byte) error {
	path, err := git.ObjectPathFromHash(objectsDir, hash[:])
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return nil
}

type treeEntry struct {
	Mode string
	Name string
	Hash [sha1.Size]byte
}

func parseTreeEntries(data []byte) ([]treeEntry, error) {
	var entries []treeEntry
	buf := data
	for len(buf) > 0 {
		space := bytes.IndexByte(buf, ' ')
		if space < 0 {
			return nil, errors.New("invalid tree entry: missing mode delimiter")
		}
		mode := string(buf[:space])
		buf = buf[space+1:]

		nul := bytes.IndexByte(buf, 0)
		if nul < 0 {
			return nil, errors.New("invalid tree entry: missing name terminator")
		}
		name := string(buf[:nul])
		buf = buf[nul+1:]

		if len(buf) < sha1.Size {
			return nil, errors.New("invalid tree entry: truncated sha1")
		}
		var hash [sha1.Size]byte
		copy(hash[:], buf[:sha1.Size])
		buf = buf[sha1.Size:]

		entries = append(entries, treeEntry{Mode: mode, Name: name, Hash: hash})
	}
	return entries, nil
}

func treeEntryType(mode string) git.ObjectType {
	switch mode {
	case "40000":
		return git.ObjectTypeTree
	case "160000":
		return git.ObjectTypeCommit
	default:
		return git.ObjectTypeBlob
	}
}
