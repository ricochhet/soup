package pe

import (
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	COFFStartBytesLen = 4
	COFFHeaderSize    = 20
)

const (
	DataDirSize      = 128
	DataDirEntrySize = 8
)

const (
	SH32EntrySize           = 64
	SH32NameSize            = 8
	SH32CharacteristicsSize = 4
)

var (
	COFFStartBytes = []byte{0x50, 0x45, 0x00, 0x00}
	OH64ByteSize   = binary.Size(OptionalHeader64X110{})
	SH32ByteSize   = binary.Size(SectionHeader32X28{})
)

type (
	PE     struct{}
	PEFile struct {
		Bytes []byte
		PE    pe.File
	}
)

type Section struct {
	ContentID   string
	OEP         uint64
	EncBlocks   []EncBlock
	ImageBase   uint64
	SizeOfImage uint32
	ImportDir   DataDir
	IATDir      DataDir
	RelocDir    DataDir
}

type Import struct {
	Characteristics uint32
	Timedatestamp   uint32
	ForwarderChain  uint32
	Name            uint32
	FThunk          uint32
}

type Thunk struct {
	Function uint32
	DataAddr uint32
}

type DataDir struct {
	VA   uint32
	Size uint32
}

type EncBlock struct {
	VA          uint32
	RawSize     uint32
	VirtualSize uint32
	Unk         uint32
	CRC         uint32
	Unk2        uint32
	CRC2        uint32
	Pad         uint32
	FileOffset  uint32
	Pad2        uint64
	Pad3        uint32
}

type OptionalHeader64X110 struct {
	MajorLinkerVersion          uint8
	MinorLinkerVersion          uint8
	SizeOfCode                  uint32
	SizeOfInitializedData       uint32
	SizeOfUninitializedData     uint32
	AddressOfEntryPoint         uint32
	BaseOfCode                  uint32
	ImageBase                   uint64
	SectionAlignment            uint32
	FileAlignment               uint32
	MajorOperatingSystemVersion uint16
	MinorOperatingSystemVersion uint16
	MajorImageVersion           uint16
	MinorImageVersion           uint16
	MajorSubsystemVersion       uint16
	MinorSubsystemVersion       uint16
	Win32VersionValue           uint32
	SizeOfImage                 uint32
	SizeOfHeaders               uint32
	CheckSum                    uint32
	Subsystem                   uint16
	DllCharacteristics          uint16
	SizeOfStackReserve          uint64
	SizeOfStackCommit           uint64
	SizeOfHeapReserve           uint64
	SizeOfHeapCommit            uint64
	LoaderFlags                 uint32
	NumberOfRvaAndSizes         uint32
}

type SectionHeader32X28 struct {
	VirtualSize          uint32
	VirtualAddress       uint32
	SizeOfRawData        uint32
	PointerToRawData     uint32
	PointerToRelocations uint32
	PointerToLineNumbers uint32
	NumberOfRelocations  uint16
	NumberOfLineNumbers  uint16
}

func (p *PE) Open(path string) (*PEFile, error) {
	data := &PEFile{}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	peFile, err := pe.NewFile(file)
	if err != nil {
		file.Close()
		return nil, err
	}

	b, err := io.ReadAll(file)
	if err != nil {
		file.Close()
		return nil, err
	}

	data.Bytes = b
	data.PE = *peFile

	return data, nil
}

func (p *PE) COFFHeaderOffset(s []byte) (int, error) {
	off, err := Find(s, COFFStartBytes)
	if err != nil {
		return -1, err
	}

	return off, nil
}

func (p *PE) DDBytes(s []byte) ([]byte, error) {
	off, err := p.COFFHeaderOffset(s)
	if err != nil {
		return nil, err
	}

	start := off + COFFStartBytesLen + COFFHeaderSize + OH64ByteSize
	end := off + COFFStartBytesLen + COFFHeaderSize + OH64ByteSize + DataDirSize

	return s[start:end], nil
}

func (p *PE) DDEntryOffset(s []byte, addr, size uint32) (int, error) {
	dir, err := p.DDBytes(s)
	if err != nil {
		return -1, err
	}

	b := make([]byte, DataDirEntrySize)
	binary.LittleEndian.PutUint32(b[:4], addr)
	binary.LittleEndian.PutUint32(b[4:], size)
	rva, err := Find(dir, b)

	if err != nil || rva == -1 {
		if err == nil {
			return -1, errors.New("rva is -1")
		}

		return -1, err
	}

	off, err := p.COFFHeaderOffset(s)
	if err != nil {
		return -1, err
	}

	return off + COFFStartBytesLen + COFFHeaderSize + OH64ByteSize + rva, nil
}

func (p *PE) SHSize(file pe.File) (int, error) {
	size := len(file.Sections) * SH32EntrySize

	if size == 0 {
		return -1, errors.New("section header size is 0")
	}

	return size, nil
}

func (p *PE) SHBytes(s []byte, size int) ([]byte, error) {
	off, err := p.COFFHeaderOffset(s)
	if err != nil {
		return nil, err
	}

	idx := off + COFFStartBytesLen + COFFHeaderSize + OH64ByteSize + DataDirSize

	return s[idx : idx+size], nil
}

func (p *PE) SHEntryOffset(s []byte, address int) (int, error) {
	off, err := p.COFFHeaderOffset(s)
	if err != nil {
		return -1, err
	}

	return off + COFFStartBytesLen + COFFHeaderSize + OH64ByteSize + DataDirSize + address, nil
}

func (p *PE) SectionBytes(file *PEFile, sectionVirtualAddress, sectionSize uint32) ([]byte, error) {
	var s *pe.Section

	for _, section := range file.PE.Sections {
		if sectionVirtualAddress >= section.VirtualAddress &&
			sectionVirtualAddress < section.VirtualAddress+section.Size {
			s = section
			break
		}
	}

	if s == nil {
		return nil, errors.New("section is nil")
	}

	off := sectionVirtualAddress - s.VirtualAddress + s.Offset

	return file.Bytes[off : off+sectionSize], nil
}

func (p *PE) Import(reader io.Reader) (Import, error) {
	var d Import
	return d, binary.Read(reader, binary.LittleEndian, &d)
}

func (p *PE) Thunk(reader io.Reader) (Thunk, error) {
	var d Thunk

	return d, binary.Read(reader, binary.LittleEndian, &d)
}

func (p *PE) DataDir(reader io.Reader) (DataDir, error) {
	var d DataDir

	return d, binary.Read(reader, binary.LittleEndian, &d)
}

func (p *PE) EncBlock(reader io.Reader) (EncBlock, error) {
	var d EncBlock

	return d, binary.Read(reader, binary.LittleEndian, &d)
}

func Find(data, pattern []byte) (int, error) {
	for i := range data[:len(data)-len(pattern)+1] {
		if Match(data[i:i+len(pattern)], pattern) {
			return i, nil
		}
	}

	return -1, fmt.Errorf("no matches found with pattern: %v", pattern)
}

func Match(a, b []byte) bool {
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
