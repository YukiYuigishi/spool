package git

type IndexHeader struct {
	Signature          [4]byte // 4-byte signature DIRC
	Version            uint32  // 4-byte version number: 2, 3, and 4.
	IndexEntrie        uint32  // 32-bit number of index entries.
	ExtensionSignature [4]byte // 4-byte extension signature
	Extensions         uint32  // 32-bit sizeof the extension
}

type IndexEntry struct {
	CtimeSec     uint32
	CtimeNanoSec uint32
	MtimeSec     uint32
	MtimeNano    uint32
	Dev          uint32
	Ino          uint32
	Mode         uint32
}
