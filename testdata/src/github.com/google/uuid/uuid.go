// Package uuid is a minimal stub of github.com/google/uuid for tests.
package uuid

import (
	"database/sql/driver"
	"hash"
)

type UUID [16]byte

type Version byte

var (
	Nil          UUID
	Max          = UUID{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	NameSpaceDNS UUID
)

func New() UUID                                 { return UUID{} }
func NewString() string                         { return "" }
func NewRandom() (UUID, error)                  { return UUID{}, nil }
func NewV7() (UUID, error)                      { return UUID{}, nil }
func NewSHA1(space UUID, data []byte) UUID      { return UUID{} }
func NewHash(h hash.Hash, s UUID, d []byte, v int) UUID { return UUID{} }
func Parse(s string) (UUID, error)              { return UUID{}, nil }
func MustParse(s string) UUID                   { return UUID{} }
func Must(u UUID, err error) UUID               { return u }

func (u UUID) String() string                   { return "" }
func (u UUID) MarshalText() ([]byte, error)     { return nil, nil }
func (u *UUID) UnmarshalText(b []byte) error    { return nil }
func (u UUID) Version() Version                 { return 0 }

func (u UUID) Value() (driver.Value, error)     { return nil, nil }
func (u UUID) MarshalBinary() ([]byte, error)   { return nil, nil }
