package json

import (
	"encoding/json"
	"os"
	"sync/atomic"
)

type (
	AtomicBool    struct{ atomic.Bool }
	AtomicInt32   struct{ atomic.Int32 }
	AtomicInt64   struct{ atomic.Int64 }
	AtomicUint32  struct{ atomic.Uint32 }
	AtomicUint64  struct{ atomic.Uint64 }
	AtomicUintptr struct{ atomic.Uintptr }
)

func (b *AtomicBool) MarshalJSON() ([]byte, error)       { return Marshal(b.Load()) }
func (b *AtomicBool) UnmarshalJSON(data []byte) error    { return UnmarshalAndStore(data, b.Store) }
func (i *AtomicInt32) MarshalJSON() ([]byte, error)      { return Marshal(i.Load()) }
func (i *AtomicInt32) UnmarshalJSON(data []byte) error   { return UnmarshalAndStore(data, i.Store) }
func (i *AtomicInt64) MarshalJSON() ([]byte, error)      { return Marshal(i.Load()) }
func (i *AtomicInt64) UnmarshalJSON(data []byte) error   { return UnmarshalAndStore(data, i.Store) }
func (u *AtomicUint32) MarshalJSON() ([]byte, error)     { return Marshal(u.Load()) }
func (u *AtomicUint32) UnmarshalJSON(data []byte) error  { return UnmarshalAndStore(data, u.Store) }
func (u *AtomicUint64) MarshalJSON() ([]byte, error)     { return Marshal(u.Load()) }
func (u *AtomicUint64) UnmarshalJSON(data []byte) error  { return UnmarshalAndStore(data, u.Store) }
func (u *AtomicUintptr) MarshalJSON() ([]byte, error)    { return Marshal(u.Load()) }
func (u *AtomicUintptr) UnmarshalJSON(data []byte) error { return UnmarshalAndStore(data, u.Store) }

func Write(name string, data []byte) error { return os.WriteFile(name, data, 0o644) }
func Marshal[T any](v T) ([]byte, error)   { return json.Marshal(v) }
func Unmarshal[T any](data func() []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(data(), &v); err != nil {
		return nil, err
	}

	return &v, nil
}

func UnmarshalAndStore[T any](data []byte, store func(T)) error {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	store(v)

	return nil
}

func ReadAndUnmarshal[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return Unmarshal[T](func() []byte {
		return data
	})
}

func MarshalAndWrite[T any](path string, data T) ([]byte, error) {
	json, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return nil, err
	}

	return json, Write(path, json)
}
