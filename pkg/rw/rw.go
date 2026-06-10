package rw

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf16"
	"unsafe"
)

const (
	I8  = 1
	U8  = 1
	I16 = 2
	U16 = 2
	I32 = 4
	U32 = 4
	I64 = 8
	U64 = 8
)

type Reader struct {
	b   []byte
	pos int
}

type Writer struct {
	b   []byte
	pos int
}

type Integer interface {
	~int | ~int8 | ~uint8 | ~int16 | ~uint16 | ~int32 | ~uint32 | ~int64 | ~uint64
}

func NewReader(b []byte) *Reader { return &Reader{b: b} }
func NewWriter(b []byte) *Writer { return &Writer{b: b} }
func (r *Reader) Bytes() []byte  { return r.b }
func (w *Writer) Bytes() []byte  { return w.b }
func (r *Reader) Pos() int       { return r.pos }
func (w *Writer) Pos() int       { return w.pos }

func (r *Reader) U8() uint8 {
	v := r.b[r.pos]
	r.pos += U8

	return v
}

func (w *Writer) U8(v uint8) {
	w.b[w.pos] = v
	w.pos += U8
}

func (r *Reader) U16() uint16 {
	v := binary.LittleEndian.Uint16(r.b[r.pos:])
	r.pos += U16

	return v
}

func (w *Writer) U16(v uint16) {
	binary.LittleEndian.PutUint16(w.b[w.pos:], v)
	w.pos += U16
}

func (r *Reader) U32() uint32 {
	v := binary.LittleEndian.Uint32(r.b[r.pos:])
	r.pos += U32

	return v
}

func (w *Writer) U32(v uint32) {
	binary.LittleEndian.PutUint32(w.b[w.pos:], v)
	w.pos += U32
}

func (r *Reader) U64() uint64 {
	v := binary.LittleEndian.Uint64(r.b[r.pos:])
	r.pos += U64

	return v
}

func (w *Writer) U64(v uint64) {
	binary.LittleEndian.PutUint64(w.b[w.pos:], v)
	w.pos += U64
}

func (r *Reader) I16() int16     { return int16(r.U16()) }
func (w *Writer) I16(v int16)    { w.U16(uint16(v)) }
func (r *Reader) I8() int8       { return int8(r.U8()) }
func (w *Writer) I8(v int8)      { w.U8(uint8(v)) }
func (r *Reader) I32() int32     { return int32(r.U32()) }
func (w *Writer) I32(v int32)    { w.U32(uint32(v)) }
func (r *Reader) I64() int64     { return int64(r.U64()) }
func (w *Writer) I64(v int64)    { w.U64(uint64(v)) }
func (r *Reader) BoolU8() bool   { return r.U8() != 0 }
func (w *Writer) BoolU8(v bool)  { w.U8(boolv[uint8](v)) }
func (r *Reader) BoolI8() bool   { return r.I8() != 0 }
func (w *Writer) BoolI8(v bool)  { w.I8(boolv[int8](v)) }
func (r *Reader) BoolU16() bool  { return r.U16() != 0 }
func (w *Writer) BoolU16(v bool) { w.U16(boolv[uint16](v)) }
func (r *Reader) BoolI16() bool  { return r.I16() != 0 }
func (w *Writer) BoolI16(v bool) { w.I16(boolv[int16](v)) }
func (r *Reader) BoolU32() bool  { return r.U32() != 0 }
func (w *Writer) BoolU32(v bool) { w.U32(boolv[uint32](v)) }
func (r *Reader) BoolI32() bool  { return r.I32() != 0 }
func (w *Writer) BoolI32(v bool) { w.I32(boolv[int32](v)) }
func (r *Reader) BoolU64() bool  { return r.U64() != 0 }
func (w *Writer) BoolU64(v bool) { w.U64(boolv[uint64](v)) }
func (r *Reader) BoolI64() bool  { return r.I64() != 0 }
func (w *Writer) BoolI64(v bool) { w.I64(boolv[int64](v)) }

func (r *Reader) UTF16(charCount int) string {
	u16 := make([]uint16, charCount)
	for i := range u16 {
		u16[i] = r.U16()
	}

	return string(utf16.Decode(u16))
}

func (w *Writer) UTF16(s string) {
	for _, u := range utf16.Encode([]rune(s)) {
		w.U16(u)
	}
}

func UTF16(value []byte, start, end int) string {
	var cid string

	if end > len(value) {
		end = len(value)
	}

	raw := value[start:end]
	u16 := (*[1 << 30]uint16)(unsafe.Pointer(&raw[0]))[:len(raw)/2]
	ind := -1

	for i, c := range u16 {
		if c == 0 {
			ind = i
			break
		}
	}

	if ind != -1 {
		cid = string(utf16.Decode(u16[:ind]))
	} else {
		cid = string(utf16.Decode(u16))
	}

	return cid
}

func UTF8ToUTF16(value string) []byte {
	b := []byte(value)
	r := utf16.Encode([]rune(string(b)))
	u16b := make([]byte, len(r)*2)

	for i, r := range r {
		u16b[i*2] = byte(r)
		u16b[i*2+1] = byte(r >> 8)
	}

	return u16b
}

func RLE[T Integer](b bytes.Buffer) T {
	var v T

	_ = binary.Read(&b, binary.LittleEndian, &v)

	return v
}

func RBE[T Integer](b bytes.Buffer) T {
	var v T

	_ = binary.Read(&b, binary.BigEndian, &v)

	return v
}

func WLE[T Integer](b *bytes.Buffer, v T) { _ = binary.Write(b, binary.LittleEndian, v) }
func WBE[T Integer](b *bytes.Buffer, v T) { _ = binary.Write(b, binary.BigEndian, v) }

func Write(s []byte, offset int, replace []byte) error {
	if offset < 0 || offset+len(replace) > len(s) {
		return fmt.Errorf("invalid offset or byte range: %d", offset)
	}

	copy(s[offset:], replace)

	return nil
}

func Bytes(v string) ([]byte, error) {
	var data []byte

	for i := 0; i < len(v); i += 2 {
		var b byte

		if _, err := fmt.Sscanf(v[i:i+2], "%02X", &b); err != nil {
			return nil, err
		}

		data = append(data, b)
	}

	return data, nil
}

func Replace(data, old, replace []byte, index int) []byte {
	var (
		result []byte
		cur    int
	)

	for r := data; len(r) > 0; cur++ {
		bi := bytes.Index(r, old)
		if bi == -1 {
			result = append(result, r...)
			break
		}

		result = append(result, r[:bi]...)

		if index == 0 || cur == index {
			tex := min(len(replace), len(old))
			result = append(result, replace[:tex]...)
		} else {
			result = append(result, old...)
		}

		r = r[bi+len(old):]
	}

	return result
}

func Pad(s []byte, size int) []byte {
	if len(s) < size {
		ps := size - len(s)
		p := make([]byte, ps)

		return append(s, p...)
	}

	return s
}

func Find(data, pattern []byte) (int, error) {
	for i := range data[:len(data)-len(pattern)+1] {
		if Match(data[i:i+len(pattern)], pattern) {
			return i, nil
		}
	}

	return -1, fmt.Errorf("no matches found with pattern: %v", pattern)
}

func FindAll(data, pattern []byte) []int {
	var idx []int

	for i := range data {
		if bytes.HasPrefix(data[i:], pattern) {
			idx = append(idx, i)
		}
	}

	return idx
}

func Match(a, b []byte) bool {
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func Swp(a, b uint16) uint16    { return (a >> b) | (a << b) }
func Swpx(v uint16) uint16      { return (v >> 8) | (v << 8) }
func Ror(v, r, m uint16) uint16 { r %= m; return (v >> r) | (v << (m - r)) }
func Rorx(v, r uint16) uint16   { r %= 16; return (v >> r) | (v << (16 - r)) }
func Rol(v, r, m uint16) uint16 { r %= m; return (v << r) | (v >> (m - r)) }
func Rolx(v, r uint16) uint16   { r %= 16; return (v << r) | (v >> (16 - r)) }

func Rot(i, a, o, m int) uint16 {
	b := -(i & a)

	r := (-o - b) % m
	if r < 0 {
		r += m
	}

	return uint16(r)
}

func Rotx(i int) uint16 {
	b := -(i & 7)

	r := (-11 - b) % 16
	if r < 0 {
		r += 16
	}

	return uint16(r)
}

func boolv[T Integer](v bool) T {
	if v {
		return 1
	}

	return 0
}
