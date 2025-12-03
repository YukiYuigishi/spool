package git

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ObjectType string

const (
	ObjectTypeBlob   ObjectType = "blob"
	ObjectTypeTree   ObjectType = "tree"
	ObjectTypeCommit ObjectType = "commit"
	ObjectTypeTag    ObjectType = "tag"
)

var validObjectTypes = map[ObjectType]struct{}{
	ObjectTypeBlob:   {},
	ObjectTypeTree:   {},
	ObjectTypeCommit: {},
	ObjectTypeTag:    {},
}

type Object struct {
	Type ObjectType
	Hash [sha1.Size]byte
	Path string
	Size int64
}

func (t ObjectType) IsValid() bool {
	_, ok := validObjectTypes[t]
	return ok
}

// ParseObjectType converts a raw string into a supported object type.
func ParseObjectType(raw string) (ObjectType, error) {
	t := ObjectType(strings.ToLower(raw))
	if !t.IsValid() {
		return "", fmt.Errorf("unsupported object type %q", raw)
	}
	return t, nil
}

// HashObject computes the Git object id for the provided payload.
func HashObject(src io.ReadSeeker, typ ObjectType, size int64) ([sha1.Size]byte, error) {
	var sum [sha1.Size]byte

	if !typ.IsValid() {
		return sum, fmt.Errorf("unsupported object type %q", typ)
	}

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return sum, err
	}

	header := formatObjectHeader(typ, size)
	h := sha1.New()

	if _, err := h.Write([]byte(header)); err != nil {
		return sum, err
	}

	if _, err := io.Copy(h, src); err != nil {
		return sum, err
	}

	copy(sum[:], h.Sum(nil))

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return sum, err
	}

	return sum, nil
}

// StoreObject persists a Git object in the objects directory.
func StoreObject(objectsDir string, typ ObjectType, size int64, src io.ReadSeeker) ([sha1.Size]byte, string, error) {
	hash, err := HashObject(src, typ, size)
	if err != nil {
		return hash, "", err
	}

	objectPath, err := ObjectPathFromHash(objectsDir, hash[:])
	if err != nil {
		return hash, "", err
	}

	if _, statErr := os.Stat(objectPath); statErr == nil {
		return hash, objectPath, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return hash, "", statErr
	}

	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		return hash, "", err
	}

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return hash, "", err
	}

	objFile, err := os.Create(objectPath)
	if err != nil {
		return hash, "", err
	}

	zw := zlib.NewWriter(objFile)
	header := formatObjectHeader(typ, size)

	if _, err := zw.Write([]byte(header)); err != nil {
		zw.Close()
		objFile.Close()
		return hash, "", err
	}

	if _, err := io.Copy(zw, src); err != nil {
		zw.Close()
		objFile.Close()
		return hash, "", err
	}

	if err := zw.Close(); err != nil {
		objFile.Close()
		return hash, "", err
	}

	if err := objFile.Close(); err != nil {
		return hash, "", err
	}

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return hash, objectPath, err
	}

	return hash, objectPath, nil
}

// ObjectPathFromHash returns the on-disk path for an object hash.
func ObjectPathFromHash(objectsDir string, hash []byte) (string, error) {
	if len(hash) != sha1.Size {
		return "", fmt.Errorf("invalid hash length: %d", len(hash))
	}

	hexHash := hex.EncodeToString(hash)
	return filepath.Join(objectsDir, hexHash[:2], hexHash[2:]), nil
}

func formatObjectHeader(typ ObjectType, size int64) string {
	return fmt.Sprintf("%s %d\x00", typ, size)
}

// ReadObject loads and decompresses the Git object addressed by hash.
func ReadObject(objectsDir string, hash []byte) (Object, []byte, error) {
	var obj Object

	if len(hash) != sha1.Size {
		return obj, nil, fmt.Errorf("invalid hash length: %d", len(hash))
	}

	path, err := ObjectPathFromHash(objectsDir, hash)
	if err != nil {
		return obj, nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return obj, nil, err
	}
	defer file.Close()

	zr, err := zlib.NewReader(file)
	if err != nil {
		return obj, nil, err
	}
	defer zr.Close()

	payload, err := io.ReadAll(zr)
	if err != nil {
		return obj, nil, err
	}

	headerEnd := bytes.IndexByte(payload, 0)
	if headerEnd < 0 {
		return obj, nil, errors.New("invalid git object: missing header terminator")
	}

	header := string(payload[:headerEnd])
	body := payload[headerEnd+1:]

	objType, size, err := parseObjectHeader(header)
	if err != nil {
		return obj, nil, err
	}

	if int64(len(body)) != size {
		return obj, nil, fmt.Errorf("object size mismatch: expected %d, got %d", size, len(body))
	}

	obj = Object{
		Type: objType,
		Path: path,
		Size: size,
	}
	copy(obj.Hash[:], hash)

	return obj, body, nil
}

func parseObjectHeader(header string) (ObjectType, int64, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid object header: %q", header)
	}

	size, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid size in header: %w", err)
	}

	objType := ObjectType(parts[0])
	if !objType.IsValid() {
		return "", 0, fmt.Errorf("unknown object type %q", parts[0])
	}

	return objType, size, nil
}

// ParseHash converts a 40-character hexadecimal object id into raw bytes.
func ParseHash(hexHash string) ([sha1.Size]byte, error) {
	var hash [sha1.Size]byte
	if len(hexHash) != sha1.Size*2 {
		return hash, fmt.Errorf("invalid hash length: %d", len(hexHash))
	}
	decoded, err := hex.DecodeString(strings.ToLower(hexHash))
	if err != nil {
		return hash, err
	}
	copy(hash[:], decoded)
	return hash, nil
}

// ExpandHashPrefix resolves a short object id prefix to its full hash.
func ExpandHashPrefix(objectsDir, prefix string) ([sha1.Size]byte, error) {
	var hash [sha1.Size]byte
	prefix = strings.ToLower(prefix)

	switch {
	case len(prefix) == sha1.Size*2:
		return ParseHash(prefix)
	case len(prefix) < 2:
		return hash, fmt.Errorf("object prefix too short: %q", prefix)
	}

	dir := filepath.Join(objectsDir, prefix[:2])
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return hash, fmt.Errorf("object not found for prefix %q", prefix)
		}
		return hash, err
	}

	rest := prefix[2:]
	var matches []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.HasPrefix(name, rest) {
			matches = append(matches, name)
		}
	}

	switch len(matches) {
	case 0:
		return hash, fmt.Errorf("object not found for prefix %q", prefix)
	case 1:
		return ParseHash(prefix[:2] + matches[0])
	default:
		return hash, fmt.Errorf("ambiguous object prefix %q", prefix)
	}
}
