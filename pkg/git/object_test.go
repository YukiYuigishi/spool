package git

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestParseObjectType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    ObjectType
		wantErr bool
	}{
		{name: "blob", input: "blob", want: ObjectTypeBlob},
		{name: "Tree case-insensitive", input: "TrEe", want: ObjectTypeTree},
		{name: "commit", input: "commit", want: ObjectTypeCommit},
		{name: "tag", input: "tag", want: ObjectTypeTag},
		{name: "invalid", input: "weird", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseObjectType(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestHashObject(t *testing.T) {
	t.Parallel()

	data := []byte("hello world\n")
	reader := bytes.NewReader(data)

	got, err := HashObject(reader, ObjectTypeBlob, int64(len(data)))
	if err != nil {
		t.Fatalf("HashObject returned error: %v", err)
	}

	expected := sha1.Sum(append([]byte("blob 12\x00"), data...))
	if got != expected {
		t.Fatalf("unexpected hash, want %s got %s", hex.EncodeToString(expected[:]), hex.EncodeToString(got[:]))
	}
}

func TestHashObjectInvalidType(t *testing.T) {
	t.Parallel()

	reader := bytes.NewReader([]byte("data"))
	if _, err := HashObject(reader, ObjectType("unknown"), 4); err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}

func TestStoreAndReadObject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	objectsDir := filepath.Join(dir, "objects")
	if err := os.MkdirAll(objectsDir, 0o755); err != nil {
		t.Fatalf("failed to create objects dir: %v", err)
	}

	content := []byte("sample blob\n")
	reader := bytes.NewReader(content)

	hash, path, err := StoreObject(objectsDir, ObjectTypeBlob, int64(len(content)), reader)
	if err != nil {
		t.Fatalf("StoreObject failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stored object missing: %v", err)
	}

	obj, body, err := ReadObject(objectsDir, hash[:])
	if err != nil {
		t.Fatalf("ReadObject failed: %v", err)
	}

	if obj.Type != ObjectTypeBlob {
		t.Fatalf("expected blob type, got %s", obj.Type)
	}
	if obj.Size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), obj.Size)
	}
	if !bytes.Equal(body, content) {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestStoreObjectDeterminism(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	objectsDir := filepath.Join(dir, "objects")
	if err := os.MkdirAll(objectsDir, 0o755); err != nil {
		t.Fatalf("failed to create objects dir: %v", err)
	}

	content := []byte("deterministic\n")
	reader := bytes.NewReader(content)

	hash1, path1, err := StoreObject(objectsDir, ObjectTypeBlob, int64(len(content)), reader)
	if err != nil {
		t.Fatalf("StoreObject failed: %v", err)
	}

	reader.Reset(content)
	hash2, path2, err := StoreObject(objectsDir, ObjectTypeBlob, int64(len(content)), reader)
	if err != nil {
		t.Fatalf("StoreObject second call failed: %v", err)
	}

	if hash1 != hash2 {
		t.Fatalf("hash mismatch between identical inputs")
	}
	if path1 != path2 {
		t.Fatalf("path mismatch between identical inputs")
	}
}

func TestExpandHashPrefix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	objectsDir := filepath.Join(dir, "objects")
	if err := os.MkdirAll(objectsDir, 0o755); err != nil {
		t.Fatalf("failed to create objects dir: %v", err)
	}

	content := []byte("prefix test\n")
	reader := bytes.NewReader(content)

	hash, _, err := StoreObject(objectsDir, ObjectTypeBlob, int64(len(content)), reader)
	if err != nil {
		t.Fatalf("StoreObject failed: %v", err)
	}
	full := hex.EncodeToString(hash[:])

	// exact match
	got, err := ExpandHashPrefix(objectsDir, full)
	if err != nil {
		t.Fatalf("ExpandHashPrefix failed for full hash: %v", err)
	}
	if got != hash {
		t.Fatalf("expected %s, got %s", full, hex.EncodeToString(got[:]))
	}

	// short prefix resolve
	prefix := full[:8]
	got, err = ExpandHashPrefix(objectsDir, prefix)
	if err != nil {
		t.Fatalf("ExpandHashPrefix failed for prefix: %v", err)
	}
	if got != hash {
		t.Fatalf("expected hash %s for prefix %s", full, prefix)
	}

	// not found
	if _, err := ExpandHashPrefix(objectsDir, "deadbeef"); err == nil {
		t.Fatalf("expected error for unknown prefix")
	}

	// ambiguous
	subdir := filepath.Join(objectsDir, full[:2])
	otherName := "0123456789abcdef0123456789abcdef0123"
	if err := os.WriteFile(filepath.Join(subdir, otherName), []byte("dummy"), 0o644); err != nil {
		t.Fatalf("failed to create dummy object: %v", err)
	}
	prefix = full[:4]
	if _, err := ExpandHashPrefix(objectsDir, prefix); err == nil {
		t.Fatalf("expected ambiguity error")
	}
}

func TestParseHash(t *testing.T) {
	t.Parallel()

	data := sha1.Sum([]byte("hash"))
	hexHash := hex.EncodeToString(data[:])

	got, err := ParseHash(hexHash)
	if err != nil {
		t.Fatalf("ParseHash failed: %v", err)
	}
	if got != data {
		t.Fatalf("expected %s, got %s", hexHash, hex.EncodeToString(got[:]))
	}

	if _, err := ParseHash("abcd"); err == nil {
		t.Fatalf("expected error for short hash")
	}
}

func TestExpandHashPrefixTooShort(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	objectsDir := filepath.Join(dir, "objects")
	if _, err := ExpandHashPrefix(objectsDir, "a"); err == nil {
		t.Fatalf("expected error for prefix shorter than two characters")
	}
}

func TestReadObjectInvalidHeader(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	objectsDir := filepath.Join(dir, "objects")
	subdir := filepath.Join(objectsDir, "aa")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	var hash [sha1.Size]byte
	copy(hash[:], []byte{0xaa, 0xbb})
	name := hex.EncodeToString(hash[:])
	path := filepath.Join(subdir, name[2:])

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create object file: %v", err)
	}

	zw := zlib.NewWriter(file)
	if _, err := zw.Write([]byte("invalid header")); err != nil {
		t.Fatalf("failed to write invalid payload: %v", err)
	}
	zw.Close()
	file.Close()

	if _, _, err := ReadObject(objectsDir, hash[:]); err == nil {
		t.Fatalf("expected error for malformed object")
	}
}
