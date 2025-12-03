package cmd

import (
    "bytes"
    "crypto/sha1"
    "testing"
)

func TestParseTreeEntries(t *testing.T) {
    t.Parallel()

    entry1 := buildTreeEntry("100644", "file.txt", bytes.Repeat([]byte{0x01}, sha1.Size))
    entry2 := buildTreeEntry("40000", "dir", bytes.Repeat([]byte{0x02}, sha1.Size))

    entries, err := parseTreeEntries(append(entry1, entry2...))
    if err != nil {
        t.Fatalf("parseTreeEntries returned error: %v", err)
    }
    if len(entries) != 2 {
        t.Fatalf("expected 2 entries, got %d", len(entries))
    }
    if entries[0].Mode != "100644" || entries[0].Name != "file.txt" {
        t.Fatalf("unexpected first entry: %+v", entries[0])
    }
    if entries[1].Mode != "40000" || entries[1].Name != "dir" {
        t.Fatalf("unexpected second entry: %+v", entries[1])
    }
}

func TestParseTreeEntriesErrors(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name string
        data []byte
    }{
        {name: "missing space", data: []byte("100644file.txt\x00" + string(make([]byte, sha1.Size)))},
        {name: "missing nul", data: append([]byte("100644 file"), bytes.Repeat([]byte{0x03}, sha1.Size)... )},
        {name: "short sha", data: append([]byte("100644 file\x00"), []byte{0x01, 0x02}...)},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            if _, err := parseTreeEntries(tt.data); err == nil {
                t.Fatalf("expected error")
            }
        })
    }
}

func TestTreeEntryType(t *testing.T) {
    t.Parallel()

    tests := []struct {
        mode string
        want string
    }{
        {mode: "40000", want: "tree"},
        {mode: "160000", want: "commit"},
        {mode: "100644", want: "blob"},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.mode, func(t *testing.T) {
            t.Parallel()
            got := treeEntryType(tt.mode)
            if string(got) != tt.want {
                t.Fatalf("treeEntryType(%s) = %s, want %s", tt.mode, got, tt.want)
            }
        })
    }
}

func buildTreeEntry(mode, name string, hash []byte) []byte {
    buf := append([]byte(mode+" "), append([]byte(name), 0)...)
    return append(buf, hash...)
}
