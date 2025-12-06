package git

type IndexHeader struct {
	Signature   [4]byte // 4-byte signature DIRC
	Version     uint32  // 4-byte version number: 2, 3, and 4.
	IndexEntries uint32  // 32-bit number of index entries.
}

// reference https://git-scm.com/docs/gitformat-index
type IndexEntry struct {
	CtimeSec     uint32
	CtimeNanoSec uint32
	MtimeSec     uint32
	MtimeNano    uint32
	Dev          uint32
	Ino          uint32
	Mode         uint32
	Reserved1    uint16 // must be zero
	// 4-bit object type 1000(regular file), 1010(symbolic link), 1110(gitlink)
	// 3-bit unused must be zero
	// 9-bit unix permission
	// |4-bit object type|3-bit zero|9-bit unix permission|
	ObjectTypeWithPermission uint16
	Uid                      uint32
	Gid                      uint32
	FileSize                 uint32
	// object name
	OID [20]byte
	// Object name for the represented object?
	// 16-bit flags
	// 1-bit assume-valid flag
	// 1-bit extended flag (must be zero in v2)
	// 2-bit stage(during merge)
	// 12-bit name length is less than 0xFFF. otherwise 0xFFF store
	Flags uint16
	// 16-bit flags
	// 1-bit reserved
	// 1-bit skip-worktree flag
	// 1-bit intent-to-add flag
	// 13-bit unused, must be zero
	ExtendedFlags uint16
}
