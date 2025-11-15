package cmd

import (
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type HashObjectOptions struct {
	UseStdin bool
	Write    bool
}

func NewHashObjectCmd() *cobra.Command {
	opts := &HashObjectOptions{}

	cmd := &cobra.Command{
		Use:   "hash-object",
		Short: "Compute an object id",
		Long:  "Compute an object id",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHashObjetCmd(cmd, opts, args)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.UseStdin, "stdin", false, "input from stdio")
	flags.BoolVar(&opts.Write, "w", false, "store object")

	return cmd
}

func runHashObjetCmd(cmd *cobra.Command, opts *HashObjectOptions, args []string) error {
	// いったんwriteのみ
	// spool hash-object -w target.txt
	if len(args) < 1 {
		return errors.New("target file is required")
	}
	objectname, err := ResolveForObject(args[0])
	if err != nil {
		return err
	}
	log.Printf("file path %s\n", objectname)

	gitroot, err := ResolveForGitRoot()
	if err != nil {
		return err
	}
	log.Printf("gitroot %s\n", gitroot)

	objectsdir := filepath.Join(gitroot, ".git", "objects")
	log.Println(objectsdir)

	f, err := os.Open(objectname)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}
	header := fmt.Sprintf("blob %d\x00", stat.Size())

	h := sha1.New()
	if _, err := h.Write([]byte(header)); err != nil {
		return err
	}

	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	hash := h.Sum(nil)
	fmt.Printf("%x\t%x/%x\n", hash, hash[:1], hash[1:])

	objectpath := filepath.Join(objectsdir, fmt.Sprintf("%x", hash[:1]), fmt.Sprintf("%x", hash[1:]))
	log.Println(objectpath)
	objdir := filepath.Join(objectsdir, hex.EncodeToString(hash[:1]))
	log.Printf("object directory; %s\n", objdir)

	// mkdir gitroot/.git/objects/hash[:1]
	err = os.Mkdir(objdir, 0755)
	log.Println(err)
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	gitobject := filepath.Join(objdir, hex.EncodeToString(hash[1:]))
	// ファイルが存在している場合は終了
	_, err = os.Stat(gitobject)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	objFile, err := os.Create(gitobject)
	if err != nil {
		return err
	}
	defer objFile.Close()

	zw := zlib.NewWriter(objFile)
	defer zw.Close()

	if _, err := zw.Write([]byte(header)); err != nil {
		return err
	}

	if _, err := io.Copy(zw, f); err != nil {
		return err
	}

	return nil
}

func ResolveForObject(p string) (string, error) {
	// ~ 展開を許可したいならここで書く（任意）

	// パス正規化（../ ./ を除去）
	p = filepath.Clean(p)

	// 現在の作業ディレクトリ基準で絶対パスに
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}

	// git hash-object は シンボリックリンク先を辿って中身を読むため、
	// EvalSymlinks を行うのが自然
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// ファイルが存在しない時などは abs でよい（--stdin / -tなどもあるため）
		return abs, nil
	}
	return real, nil
}

// ResolveForGitRoot は `.git` ディレクトリが存在する親ディレクトリを返す。
// start を指定した場合はそのパス（相対指定やシンボリックリンク可）から探索を開始し、
// 指定がなければ os.Getwd() を使う。これによりテストでは任意の開始地点を明示でき、
// 実行時は引数なしで自然にカレントディレクトリを基準にできる。
func ResolveForGitRoot(start ...string) (string, error) {
	var (
		dir string
		err error
	)

	if len(start) > 0 && start[0] != "" {
		dir = start[0]
	} else {
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}

	searchRoot := dir
	for {
		gitPath := filepath.Join(dir, ".git")
		_, statErr := os.Stat(gitPath)
		if statErr == nil {
			return dir, nil
		}

		if !errors.Is(statErr, os.ErrNotExist) {
			return "", statErr
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf(".git not found from %s", searchRoot)
}

type ObjectType int

const (
	ObjectBlob ObjectType = iota
	ObjectTree
)
