package vv

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var bpThresholds = [8]uint64{
	128,
	16_512,
	2_113_664,
	270_549_120,
	34_630_287_488,
	4_432_676_798_592,
	567_382_630_219_904,
	72_624_976_668_147_840,
}

var (
	TableSentinel = JOAAT("\x00owf-archive-index\x00")
	errStop       = errors.New("stop")
)

type Entry struct {
	Version uint64
	Value   uint64
}

type File[T any] struct {
	Path    T
	Content []byte
}

func JOAAT(s string) uint32 {
	var h uint32
	for i := range len(s) {
		h += uint32(s[i])
		h += h << 10
		h ^= h >> 6
	}

	h += h << 3
	h ^= h >> 11
	h += h << 15

	return h
}

func Packx(entries []File[string], prefix string, embed bool) ([]byte, []byte) {
	b := Pack(entries, prefix, embed)

	var sb strings.Builder

	for _, e := range entries {
		path := prefix + e.Path
		fmt.Fprintf(&sb, "%08x:%s\n", JOAAT(path), path)
	}

	return b, []byte(sb.String())
}

func Unpackx(data []byte, output string, flat bool) error {
	entries, err := Unpack(data)
	if err != nil {
		return err
	}

	for _, e := range entries {
		rel := filepath.FromSlash(e.Path)
		if flat {
			rel = filepath.Base(rel)
		}

		p := filepath.Join(output, rel)
		if !subpath(output, p) {
			return fmt.Errorf("path %q escapes output directory", e.Path)
		}

		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}

		if err := os.WriteFile(p, e.Content, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func Pack(entries []File[string], prefix string, embed bool) []byte {
	files := entries
	if prefix != "" {
		files = make([]File[string], len(entries))
		for i, e := range entries {
			files[i] = File[string]{
				Path:    prefix + e.Path,
				Content: e.Content,
			}
		}
	}

	var out []byte

	if embed {
		out = appendTable(out, files)
	}

	for _, e := range files {
		out = appendRawEntry(out, JOAAT(e.Path), e.Content)
	}

	return out
}

func Unpack(data []byte) ([]File[string], error) {
	m, start, err := loadTable(data)
	if err != nil {
		return nil, err
	}

	var files []File[string]

	err = scan(data, start, func(e File[uint32]) error {
		path, ok := m[e.Path]
		if !ok {
			path = fmt.Sprintf("%08x.bin", e.Path)
		}

		files = append(files, File[string]{Path: path, Content: e.Content})

		return nil
	})

	return files, err
}

func AppendEntry(dst []byte, archivePath string, content []byte) []byte {
	return appendRawEntry(dst, JOAAT(archivePath), content)
}

func Scan(data []byte, fn func(File[uint32]) error) error {
	_, start, err := loadTable(data)
	if err != nil {
		return err
	}

	return scan(data, start, func(e File[uint32]) error {
		return fn(File[uint32]{Path: e.Path, Content: e.Content})
	})
}

func Lookup(data []byte, archivePath string) ([]byte, error) {
	want := JOAAT(archivePath)

	var found []byte

	err := Scan(data, func(e File[uint32]) error {
		if e.Path == want {
			found = e.Content
			return errStop
		}

		return nil
	})

	return found, err
}

func LookupHash(data []byte, hash uint32) ([]byte, error) {
	var found []byte

	err := Scan(data, func(e File[uint32]) error {
		if e.Path == hash {
			found = e.Content
			return errStop
		}

		return nil
	})

	return found, err
}

func DecodeAll(data []byte) ([]File[uint32], error) {
	var out []File[uint32]

	err := Scan(data, func(e File[uint32]) error {
		c := make([]byte, len(e.Content))
		copy(c, e.Content)
		out = append(out, File[uint32]{Path: e.Path, Content: c})

		return nil
	})

	return out, err
}

func Index(data []byte) (map[uint32][]byte, error) {
	idx := make(map[uint32][]byte)
	err := Scan(data, func(e File[uint32]) error {
		c := make([]byte, len(e.Content))
		copy(c, e.Content)
		idx[e.Path] = c

		return nil
	})

	return idx, err
}

func V2N(ver string) uint64 {
	var n uint64

	for i := range len(ver) {
		b := ver[i]
		if b >= '0' && b <= '9' {
			n = n*10 + uint64(b-'0')
		}
	}

	return n
}

func Enc(packer func(dst []byte, v uint64) []byte, entries []Entry) []byte {
	var out []byte
	for _, e := range entries {
		out = AppendU64DynBP(out, e.Version)
		out = packer(out, e.Value)
	}

	return out
}

func EncI64(packer func(dst []byte, v int64) []byte, entries []Entry) []byte {
	var out []byte
	for _, e := range entries {
		out = AppendU64DynBP(out, e.Version)
		out = packer(out, int64(e.Value))
	}

	return out
}

func DecU64(data []byte, ver uint64) (uint64, error) {
	offset := 0
	for offset < len(data) {
		storedVer, next, err := UnpackU64DynBP(data, offset)
		if err != nil {
			return 0, err
		}

		offset = next

		val, next, err := UnpackU64DynBP(data, offset)
		if err != nil {
			return 0, err
		}

		offset = next

		if ver >= storedVer {
			return val, nil
		}
	}

	return 0, nil
}

func DecI64(data []byte, ver uint64) (int64, error) {
	offset := 0
	for offset < len(data) {
		storedVer, next, err := UnpackU64DynBP(data, offset)
		if err != nil {
			return 0, err
		}

		offset = next

		val, next, err := UnpackI64DynBP(data, offset)
		if err != nil {
			return 0, err
		}

		offset = next

		if ver >= storedVer {
			return val, nil
		}
	}

	return 0, nil
}

func AppendU64Dyn(dst []byte, v uint64) []byte {
	for range 8 {
		cur := byte(v & 0x7f)

		v >>= 7
		if v != 0 {
			dst = append(dst, cur|0x80)
		} else {
			return append(dst, cur)
		}
	}

	return append(dst, byte(v))
}

func UnpackU64Dyn(data []byte, offset int) (uint64, int, error) {
	var (
		v     uint64
		shift uint
	)

	for range 8 {
		if offset >= len(data) {
			return 0, offset, errors.New("insufficient data")
		}

		b := data[offset]
		offset++

		v |= uint64(b&0x7f) << shift
		if b>>7 == 0 {
			return v, offset, nil
		}

		shift += 7
	}

	if offset >= len(data) {
		return 0, offset, errors.New("insufficient data")
	}

	v |= uint64(data[offset]) << 56

	return v, offset + 1, nil
}

func AppendU64DynB(dst []byte, v uint64) []byte {
	for range 8 {
		cur := byte(v & 0x7f)

		v >>= 7
		if v != 0 {
			dst = append(dst, cur|0x80)
			v--
		} else {
			return append(dst, cur)
		}
	}

	return append(dst, byte(v))
}

func UnpackU64DynB(data []byte, offset int) (uint64, int, error) {
	var (
		v     uint64
		shift uint
		bias  uint64
	)

	ninthByte := true

	for range 8 {
		if offset >= len(data) {
			return 0, offset, errors.New("insufficient data")
		}

		b := data[offset]
		offset++

		v |= uint64(b&0x7f) << shift
		if b>>7 == 0 {
			ninthByte = false
			break
		}

		shift += 7
		bias += 1 << shift
	}

	if ninthByte {
		if offset >= len(data) {
			return 0, offset, errors.New("insufficient data")
		}

		v |= uint64(data[offset]) << 56
		offset++
	}

	if bias > ^uint64(0)-v {
		return 0, offset, errors.New("invalid data")
	}

	return v + bias, offset, nil
}

func AppendU64DynP(dst []byte, v uint64) []byte {
	byteLen := u64DynPByteLen(v)

	valueBits := valueBitsForLen(byteLen)
	prefixBits := uint(byteLen - 1)

	firstByte := prefixByte(prefixBits) | byte(v&valueMask(valueBits))
	v >>= valueBits

	dst = append(dst, firstByte)
	for idx := 1; idx < byteLen; idx++ {
		dst = append(dst, byte(v>>uint((idx-1)*8)))
	}

	return dst
}

func UnpackU64DynP(data []byte, offset int) (uint64, int, error) {
	if offset >= len(data) {
		return 0, offset, errors.New("insufficient data")
	}

	firstByte := data[offset]
	byteLen, valueBits := decodePrefix(firstByte)

	if offset+byteLen > len(data) {
		return 0, offset, errors.New("insufficient data")
	}

	var v uint64
	for idx := 1; idx < byteLen; idx++ {
		v |= uint64(data[offset+idx]) << uint((idx-1)*8)
	}

	v <<= valueBits
	if valueBits > 0 {
		v |= uint64(firstByte) & valueMask(valueBits)
	}

	return v, offset + byteLen, nil
}

func AppendU64DynBP(dst []byte, v uint64) []byte {
	byteLen := 1

	for _, thresh := range bpThresholds {
		if v >= thresh {
			byteLen++
		}
	}

	valueBits := valueBitsForLen(byteLen)
	prefixBits := uint(byteLen - 1)

	bias := getBias(byteLen)
	v -= bias

	firstByte := prefixByte(prefixBits) | byte(v&valueMask(valueBits))
	v >>= valueBits

	dst = append(dst, firstByte)
	for idx := 1; idx < byteLen; idx++ {
		dst = append(dst, byte(v>>uint((idx-1)*8)))
	}

	return dst
}

func UnpackU64DynBP(data []byte, offset int) (uint64, int, error) {
	if offset >= len(data) {
		return 0, offset, errors.New("insufficient data")
	}

	firstByte := data[offset]
	byteLen, valueBits := decodePrefix(firstByte)

	if offset+byteLen > len(data) {
		return 0, offset, errors.New("insufficient data")
	}

	var v uint64
	for idx := 1; idx < byteLen; idx++ {
		v |= uint64(data[offset+idx]) << uint((idx-1)*8)
	}

	v <<= valueBits
	if valueBits > 0 {
		v |= uint64(firstByte) & valueMask(valueBits)
	}

	bias := getBias(byteLen)
	if bias > ^uint64(0)-v {
		return 0, offset, errors.New("invalid data")
	}

	return v + bias, offset + byteLen, nil
}

func AppendI64DynA(dst []byte, v int64) []byte { return AppendU64Dyn(dst, signedToUnsignedA(v)) }
func UnpackI64DynA(data []byte, offset int) (int64, int, error) {
	u, offset, err := UnpackU64Dyn(data, offset)
	if err != nil {
		return 0, offset, err
	}

	return unsignedToSignedA(u), offset, nil
}

func AppendI64DynBPA(
	dst []byte,
	v int64,
) []byte {
	return AppendU64DynBP(dst, signedToUnsignedA(v))
}

func UnpackI64DynBPA(data []byte, offset int) (int64, int, error) {
	u, offset, err := UnpackU64DynBP(data, offset)
	if err != nil {
		return 0, offset, err
	}

	return unsignedToSignedA(u), offset, nil
}

func AppendI64DynB(dst []byte, v int64) []byte { return AppendU64DynB(dst, signedToUnsignedB(v)) }
func UnpackI64DynB(data []byte, offset int) (int64, int, error) {
	u, offset, err := UnpackU64DynB(data, offset)
	if err != nil {
		return 0, offset, err
	}

	return unsignedToSignedB(u), offset, nil
}

func AppendI64DynBP(dst []byte, v int64) []byte { return AppendU64DynBP(dst, signedToUnsignedB(v)) }
func UnpackI64DynBP(data []byte, offset int) (int64, int, error) {
	u, offset, err := UnpackU64DynBP(data, offset)
	if err != nil {
		return 0, offset, err
	}

	return unsignedToSignedB(u), offset, nil
}

func subpath(src, dst string) bool {
	rel, err := filepath.Rel(filepath.Clean(src), filepath.Clean(dst))
	return err == nil && !strings.HasPrefix(rel, "..")
}

func appendRawEntry(dst []byte, hash uint32, content []byte) []byte {
	dst = binary.LittleEndian.AppendUint32(dst, hash)
	dst = AppendU64DynBP(dst, uint64(len(content)))

	return append(dst, content...)
}

func appendTable(dst []byte, entries []File[string]) []byte {
	var tb []byte

	for _, e := range entries {
		tb = binary.LittleEndian.AppendUint32(tb, JOAAT(e.Path))
		tb = AppendU64DynBP(tb, uint64(len(e.Path)))
		tb = append(tb, e.Path...)
	}

	return appendRawEntry(dst, TableSentinel, tb)
}

func loadTable(data []byte) (map[uint32]string, int, error) {
	m := map[uint32]string{}

	if len(data) < 4 {
		return m, 0, nil
	}

	if binary.LittleEndian.Uint32(data) != TableSentinel {
		return m, 0, nil
	}

	e, next, err := decodeRawEntry(data, 0)
	if err != nil {
		return nil, 0, err
	}

	tb := e.Content

	off := 0
	for off < len(tb) {
		if off+4 > len(tb) {
			return nil, 0, fmt.Errorf("path table truncated at offset %d", off)
		}

		h := binary.LittleEndian.Uint32(tb[off:])
		off += 4

		pl, newOff, err := UnpackU64DynBP(tb, off)
		if err != nil {
			return nil, 0, err
		}

		off = newOff
		if off+int(pl) > len(tb) {
			return nil, 0, errors.New("path table path data truncated")
		}

		m[h] = string(tb[off : off+int(pl)])
		off += int(pl)
	}

	return m, next, nil
}

func scan(data []byte, start int, fn func(File[uint32]) error) error {
	off := start
	for off < len(data) {
		e, next, err := decodeRawEntry(data, off)
		if err != nil {
			return err
		}

		if err := fn(e); err != nil {
			if errors.Is(err, errStop) {
				return nil
			}

			return err
		}

		off = next
	}

	return nil
}

func decodeRawEntry(data []byte, off int) (File[uint32], int, error) {
	if off+4 > len(data) {
		return File[uint32]{}, off, fmt.Errorf(
			"need 4 bytes for hash at offset %d",
			off,
		)
	}

	hash := binary.LittleEndian.Uint32(data[off:])
	off += 4

	cl, off, err := UnpackU64DynBP(data, off)
	if err != nil {
		return File[uint32]{}, off, err
	}

	end := off + int(cl)
	if end > len(data) {
		return File[uint32]{}, off, fmt.Errorf(
			"need %d bytes for content at offset %d, have %d",
			cl,
			off,
			len(data)-off,
		)
	}

	return File[uint32]{Path: hash, Content: data[off:end]}, end, nil
}

func u64DynPByteLen(v uint64) int {
	n := 1
	if v>>7 != 0 {
		n++
	}

	if v>>14 != 0 {
		n++
	}

	if v>>21 != 0 {
		n++
	}

	if v>>28 != 0 {
		n++
	}

	if v>>35 != 0 {
		n++
	}

	if v>>42 != 0 {
		n++
	}

	if v>>49 != 0 {
		n++
	}

	if v>>56 != 0 {
		n++
	}

	return n
}

func valueBitsForLen(byteLen int) uint {
	if byteLen < 8 {
		return uint(8 - byteLen)
	}

	return 0
}

func valueMask(n uint) uint64 {
	if n == 0 {
		return 0
	}

	return (1 << n) - 1
}

func prefixByte(prefixBits uint) byte {
	if prefixBits == 0 {
		return 0
	}

	return byte(0xff << (8 - prefixBits))
}

func decodePrefix(firstByte byte) (byteLen int, valueBits uint) {
	prefixBits := 0
	for prefixBits < 8 && (firstByte&(0x80>>uint(prefixBits))) != 0 {
		prefixBits++
	}

	byteLen = prefixBits + 1
	valueBits = valueBitsForLen(byteLen)

	return byteLen, valueBits
}

func signedToUnsignedA(v int64) uint64 {
	var (
		neg uint64
		mag uint64
	)

	if v < 0 {
		neg = 1
		mag = (^uint64(v) + 1) &^ (uint64(1) << 63)
	} else {
		mag = uint64(v)
	}

	return (neg << 6) | ((mag &^ uint64(0x3f)) << 1) | (mag & 0x3f)
}

func unsignedToSignedA(u uint64) int64 {
	neg := (u>>6)&1 != 0

	mag := ((u &^ uint64(0x7f)) >> 1) | (u & 0x3f)
	if neg {
		return int64(^(mag - 1) | (uint64(1) << 63))
	}

	return int64(mag)
}

func signedToUnsignedB(v int64) uint64 {
	var (
		neg uint64
		mag uint64
	)

	if v < 0 {
		neg = 1
		mag = ^uint64(v)
	} else {
		mag = uint64(v)
	}

	return (neg << 6) | ((mag &^ uint64(0x3f)) << 1) | (mag & 0x3f)
}

func unsignedToSignedB(u uint64) int64 {
	neg := (u>>6)&1 != 0

	mag := ((u &^ uint64(0x7f)) >> 1) | (u & 0x3f)
	if neg {
		return int64(^mag)
	}

	return int64(mag)
}

func getBias(byteLen int) uint64 {
	var bias uint64

	for byteLen > 1 {
		byteLen--
		bias = (bias + 1) << 7
	}

	return bias
}
