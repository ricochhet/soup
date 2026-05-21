package crypto

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"io"
	"os"
	"unicode/utf16"
)

type Hash128 interface {
	hash.Hash
	Sum128() []byte
}

const (
	c1X64_128        uint64 = 0x87c37b91114253d5
	c2X64_128        uint64 = 0x4cf5ad432745937f
	sizeX64_128      int    = 128
	blockSizeX64_128 int    = 16

	c1X86_32        uint32 = 0xcc9e2d51
	c2X86_32        uint32 = 0x1b873593
	sizeX86_32      int    = 32
	blockSizeX86_32 int    = 4

	c1X86_128        uint32 = 0x239b961b
	c2X86_128        uint32 = 0xab0e9789
	c3X86_128        uint32 = 0x38b34ae5
	c4X86_128        uint32 = 0xa1e38b93
	sizeX86_128      int    = 128
	blockSizeX86_128 int    = 16
)

type digestX64_128 struct {
	h1   uint64
	h2   uint64
	tlen int
	tail []byte
}

type digestX86_32 struct {
	h1   uint32
	tlen int
	tail []byte
}

type digestX86_128 struct {
	h    [4]uint32
	tlen int
	tail []byte
}

func NewMurmur3X64_128(seed int) Hash128 {
	return &digestX64_128{uint64(seed), uint64(seed), 0, nil}
}

func NewMurmur3X86_32(seed int) hash.Hash32 {
	return &digestX86_32{uint32(seed), 0, nil}
}

func NewMurmur3X86_128(seed int) Hash128 {
	useed := uint32(seed)
	s := [4]uint32{useed, useed, useed, useed}

	return &digestX86_128{s, 0, nil}
}

func NewHash(name string, hash hash.Hash) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func CatBread(name string) uint32 {
	var hash uint32 = 2166136261
	for i := range len(name) {
		hash ^= uint32(name[i])
		hash &= 0x7fffffff
		hash *= 16777619
		hash &= 0x7fffffff
	}

	return hash
}

func MD5(path string) (string, error)    { return NewHash(path, md5.New()) }
func SHA1(path string) (string, error)   { return NewHash(path, sha1.New()) }
func SHA256(path string) (string, error) { return NewHash(path, sha256.New()) }
func SHA512(path string) (string, error) { return NewHash(path, sha512.New()) }
func CRC32(path string) (string, error)  { return NewHash(path, crc32.New(crc32.IEEETable)) }

func CRC64(
	path string,
) (string, error) {
	return NewHash(path, crc64.New(crc64.MakeTable(crc32.IEEE)))
}

func Murmur3X64_128Hash(seed int, str string) uint64 {
	b := NewMurmur3X64_128(seed)
	b.Write(UTF8ToUTF16(str))

	return binary.LittleEndian.Uint64(b.Sum(nil))
}

func Murmur3X86_32Hash(seed int, str string) uint32 {
	b := NewMurmur3X86_32(seed)
	b.Write(UTF8ToUTF16(str))

	return binary.LittleEndian.Uint32(b.Sum(nil))
}

func Murmur3X86_128Hash(seed int, str string) uint32 {
	bytes := NewMurmur3X86_128(seed)
	bytes.Write(UTF8ToUTF16(str))

	return binary.LittleEndian.Uint32(bytes.Sum(nil))
}

func (m *digestX64_128) Write(p []byte) (n int, err error) {
	h1, h2 := m.h1, m.h2
	plen := len(p)
	nblocks := plen / 16

	if m.tail != nil {
		hlen := 16 - len(m.tail)
		head := p[:hlen]
		m.tail = append(m.tail, head...)
		k1 := binary.LittleEndian.Uint64(m.tail[:8])
		k2 := binary.LittleEndian.Uint64(m.tail[8:])
		h1, h2 = bodyX64_128(h1, h2, k1, k2)
		p = p[hlen:]
		m.tail = nil
	}

	for i := range nblocks {
		k1 := binary.LittleEndian.Uint64(p[(i * 16):])
		k2 := binary.LittleEndian.Uint64(p[(i*16 + 8):])
		h1, h2 = bodyX64_128(h1, h2, k1, k2)
	}

	m.h1 = h1
	m.h2 = h2

	m.tlen += plen
	if (plen & 15) != 0 {
		m.tail = p[nblocks*16:]
	}

	return plen, nil
}

func (m *digestX86_32) Write(p []byte) (n int, err error) {
	h1 := m.h1
	plen := len(p)
	nblocks := plen / 4

	if m.tail != nil {
		hlen := blockSizeX86_32 - len(m.tail)
		head := p[:hlen]
		m.tail = append(m.tail, head...)
		k1 := binary.LittleEndian.Uint32(m.tail)
		h1 = bodyX86_32(h1, k1)
		p = p[hlen:]
		m.tail = nil
	}

	for i := range nblocks {
		k1 := binary.LittleEndian.Uint32(p[(i * blockSizeX86_32):])
		h1 = bodyX86_32(h1, k1)
	}

	m.h1 = h1

	m.tlen += plen
	if (plen & 3) != 0 {
		m.tail = p[nblocks*blockSizeX86_32:]
	}

	return plen, nil
}

func (m *digestX86_128) Write(p []byte) (int, error) {
	h := m.h
	plen := len(p)
	nblocks := plen / 16

	if m.tail != nil {
		hlen := 16 - len(m.tail)
		head := p[:hlen]
		m.tail = append(m.tail, head...)
		k := [4]uint32{
			binary.LittleEndian.Uint32(m.tail[:4]),
			binary.LittleEndian.Uint32(m.tail[4:8]),
			binary.LittleEndian.Uint32(m.tail[8:12]),
			binary.LittleEndian.Uint32(m.tail[12:]),
		}
		h = bodyX86_128(h, k)
		p = p[hlen:]
		m.tail = nil
	}

	for i := range nblocks {
		k := [4]uint32{
			binary.LittleEndian.Uint32(p[(i * 16):]),
			binary.LittleEndian.Uint32(p[(i*16 + 4):]),
			binary.LittleEndian.Uint32(p[(i*16 + 8):]),
			binary.LittleEndian.Uint32(p[(i*16 + 12):]),
		}
		h = bodyX86_128(h, k)
	}

	m.h[0] = h[0]
	m.h[1] = h[1]
	m.h[2] = h[2]
	m.h[3] = h[3]

	m.tlen += plen
	if (plen & 15) != 0 {
		m.tail = p[nblocks*16:]
	}

	return plen, nil
}

func (m *digestX64_128) Sum(in []byte) []byte {
	h1, h2 := m.processTail()
	h1, h2 = final(h1, h2, uint64(m.tlen))

	return append(in,
		byte(h1>>0), byte(h1>>8), byte(h1>>16), byte(h1>>24),
		byte(h1>>32), byte(h1>>40), byte(h1>>48), byte(h1>>56),
		byte(h2>>0), byte(h2>>8), byte(h2>>16), byte(h2>>24),
		byte(h2>>32), byte(h2>>40), byte(h2>>48), byte(h2>>56),
	)
}

func (m *digestX86_32) Sum(in []byte) []byte {
	h1 := m.processTail()
	h1 ^= uint32(m.tlen)
	h1 = fmix32(h1)

	return append(in,
		byte(h1>>0), byte(h1>>8), byte(h1>>16), byte(h1>>24),
	)
}

func (m *digestX86_128) Sum(in []byte) []byte {
	h := m.processTail()
	h = finalX86_128(h, uint32(m.tlen))

	return append(in,
		byte(h[0]>>0), byte(h[0]>>8), byte(h[0]>>16), byte(h[0]>>24),
		byte(h[1]>>0), byte(h[1]>>8), byte(h[1]>>16), byte(h[1]>>24),
		byte(h[2]>>0), byte(h[2]>>8), byte(h[2]>>16), byte(h[2]>>24),
		byte(h[3]>>0), byte(h[3]>>8), byte(h[3]>>16), byte(h[3]>>24),
	)
}

func (m *digestX64_128) Sum128() []byte {
	bytes := make([]byte, 16)
	return m.Sum(bytes)
}

func (m *digestX86_32) Sum32() uint32 {
	bytes := make([]byte, 4)
	bytes = m.Sum(bytes)

	return binary.LittleEndian.Uint32(bytes)
}

func (m *digestX86_128) Sum128() []byte {
	bytes := make([]byte, 16)
	return m.Sum(bytes)
}

func (m *digestX64_128) Reset() {
	m.h1 = 0
	m.h2 = 0
	m.tlen = 0
	m.tail = nil
}

func (m *digestX86_32) Reset() {
	m.h1 = 0
	m.tlen = 0
	m.tail = nil
}

func (m *digestX86_128) Reset() {
	m.h[0] = 0
	m.h[1] = 0
	m.h[2] = 0
	m.h[3] = 0
	m.tlen = 0
	m.tail = nil
}

func (m *digestX64_128) Size() int      { return sizeX64_128 }
func (m *digestX86_32) Size() int       { return sizeX86_32 }
func (m *digestX86_128) Size() int      { return sizeX86_128 }
func (m *digestX64_128) BlockSize() int { return blockSizeX64_128 }
func (m *digestX86_32) BlockSize() int  { return blockSizeX86_32 }
func (m *digestX86_128) BlockSize() int { return blockSizeX86_128 }

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

func bodyX64_128(h1, h2, k1, k2 uint64) (uint64, uint64) {
	k1 *= c1X64_128
	k1 = (k1 << 31) | (k1 >> 33)
	k1 *= c2X64_128
	h1 ^= k1

	h1 = (h1<<27 | h1>>37)
	h1 += h2
	h1 = h1*5 + 0x52dce729

	k2 *= c2X64_128
	k2 = (k2<<33 | k2>>31)
	k2 *= c1X64_128
	h2 ^= k2

	h2 = (h2<<31 | h2>>33)
	h2 += h1
	h2 = h2*5 + 0x38495ab5

	return h1, h2
}

func bodyX86_32(h1, k1 uint32) uint32 {
	k1 *= c1X86_32
	k1 = (k1 << 15) | (k1 >> 17)
	k1 *= c2X86_32
	h1 ^= k1
	h1 = (h1 << 13) | (h1 >> 19)
	h1 = h1*5 + 0xe6546b64

	return h1
}

func bodyX86_128(h, k [4]uint32) [4]uint32 {
	k[0] *= c1X86_128
	k[0] = (k[0] << 15) | (k[0] >> 17)
	k[0] *= c2X86_128
	h[0] ^= k[0]

	h[0] = (h[0]<<19 | h[0]>>13)
	h[0] += h[1]
	h[0] = h[0]*5 + 0x561ccd1b

	k[1] *= c2X86_128
	k[1] = (k[1] << 16) | (k[1] >> 16)
	k[1] *= c3X86_128
	h[1] ^= k[1]

	h[1] = (h[1]<<17 | h[1]>>15)
	h[1] += h[2]
	h[1] = h[1]*5 + 0x0bcaa747

	k[2] *= c3X86_128
	k[2] = (k[2] << 17) | (k[2] >> 15)
	k[2] *= c4X86_128
	h[2] ^= k[2]

	h[2] = (h[2]<<15 | h[2]>>17)
	h[2] += h[3]
	h[2] = h[2]*5 + 0x96cd1c35

	k[3] *= c4X86_128
	k[3] = (k[3] << 18) | (k[3] >> 14)
	k[3] *= c1X86_128
	h[3] ^= k[3]

	h[3] = (h[3]<<13 | h[3]>>19)
	h[3] += h[0]
	h[3] = h[3]*5 + 0x32ac3b17

	return h
}

func final(h1, h2, tlen uint64) (uint64, uint64) {
	h1 ^= tlen
	h2 ^= tlen
	h1 += h2
	h2 += h1
	h1 = fmix64(h1)
	h2 = fmix64(h2)
	h1 += h2
	h2 += h1

	return h1, h2
}

func finalX86_128(h [4]uint32, tlen uint32) [4]uint32 {
	h[0] ^= tlen
	h[1] ^= tlen
	h[2] ^= tlen
	h[3] ^= tlen

	h[0] += h[1]
	h[0] += h[2]
	h[0] += h[3]
	h[1] += h[0]
	h[2] += h[0]
	h[3] += h[0]

	h[0] = fmix32(h[0])
	h[1] = fmix32(h[1])
	h[2] = fmix32(h[2])
	h[3] = fmix32(h[3])

	h[0] += h[1]
	h[0] += h[2]
	h[0] += h[3]
	h[1] += h[0]
	h[2] += h[0]
	h[3] += h[0]

	return h
}

func (m *digestX64_128) processTail() (uint64, uint64) {
	tail := m.tail
	h1, h2 := m.h1, m.h2
	k1 := uint64(0)
	k2 := uint64(0)

	switch m.tlen & 15 {
	case 15:
		k2 ^= uint64(tail[14]) << 48
		fallthrough
	case 14:
		k2 ^= uint64(tail[13]) << 40
		fallthrough
	case 13:
		k2 ^= uint64(tail[12]) << 32
		fallthrough
	case 12:
		k2 ^= uint64(tail[11]) << 24
		fallthrough
	case 11:
		k2 ^= uint64(tail[10]) << 16
		fallthrough
	case 10:
		k2 ^= uint64(tail[9]) << 8
		fallthrough
	case 9:
		k2 ^= uint64(tail[8]) << 0
		k2 *= c2X64_128
		k2 = (k2 << 33) | (k2 >> 31)
		k2 *= c1X64_128
		h2 ^= k2

		fallthrough
	case 8:
		k1 ^= uint64(tail[7]) << 56
		fallthrough
	case 7:
		k1 ^= uint64(tail[6]) << 48
		fallthrough
	case 6:
		k1 ^= uint64(tail[5]) << 40
		fallthrough
	case 5:
		k1 ^= uint64(tail[4]) << 32
		fallthrough
	case 4:
		k1 ^= uint64(tail[3]) << 24
		fallthrough
	case 3:
		k1 ^= uint64(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint64(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint64(tail[0]) << 0
		k1 *= c1X64_128
		k1 = (k1 << 31) | (k1 >> 33)
		k1 *= c2X64_128
		h1 ^= k1
	}

	return h1, h2
}

func (m *digestX86_32) processTail() uint32 {
	tail := m.tail
	h1 := m.h1
	k1 := uint32(0)

	switch m.tlen & 3 {
	case 3:
		k1 ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint32(tail[0])
		k1 *= c1X86_32
		k1 = (k1 << 15) | (k1 >> 17)
		k1 *= c2X86_32
		h1 ^= k1
	}

	return h1
}

func (m *digestX86_128) processTail() [4]uint32 {
	tail := m.tail
	h := m.h
	k1 := uint32(0)
	k2 := uint32(0)
	k3 := uint32(0)
	k4 := uint32(0)

	switch m.tlen & 15 {
	case 15:
		k4 ^= uint32(tail[14]) << 16
		fallthrough
	case 14:
		k4 ^= uint32(tail[13]) << 8
		fallthrough
	case 13:
		k4 ^= uint32(tail[12]) << 0
		k4 *= c4X86_128
		k4 = (k4 << 18) | (k4 >> 14)
		k4 *= c1X86_128
		h[3] ^= k4

		fallthrough
	case 12:
		k3 ^= uint32(tail[11]) << 24
		fallthrough
	case 11:
		k3 ^= uint32(tail[10]) << 16
		fallthrough
	case 10:
		k3 ^= uint32(tail[9]) << 8
		fallthrough
	case 9:
		k3 ^= uint32(tail[8]) << 0
		k3 *= c3X86_128
		k3 = (k3 << 17) | (k3 >> 15)
		k3 *= c4X86_128
		h[2] ^= k3

		fallthrough
	case 8:
		k2 ^= uint32(tail[7]) << 24
		fallthrough
	case 7:
		k2 ^= uint32(tail[6]) << 16
		fallthrough
	case 6:
		k2 ^= uint32(tail[5]) << 8
		fallthrough
	case 5:
		k2 ^= uint32(tail[4]) << 0
		k2 *= c2X86_128
		k2 = (k2 << 16) | (k2 >> 16)
		k2 *= c3X86_128
		h[1] ^= k2

		fallthrough
	case 4:
		k1 ^= uint32(tail[3]) << 24
		fallthrough
	case 3:
		k1 ^= uint32(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint32(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint32(tail[0]) << 0
		k1 *= c1X86_128
		k1 = (k1 << 15) | (k1 >> 17)
		k1 *= c2X86_128
		h[0] ^= k1
	}

	return h
}

func fmix64(k uint64) uint64 {
	k ^= k >> 33
	k *= 0xff51afd7ed558ccd
	k ^= k >> 33
	k *= 0xc4ceb9fe1a85ec53
	k ^= k >> 33

	return k
}

func fmix32(h uint32) uint32 {
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16

	return h
}
