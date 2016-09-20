// 23 october 2015
package symwave

import (
	"crypto/cipher"
	"bytes"
	"encoding/binary"

	"github.com/andlabs/reallymine/bridge"
	"github.com/mendsley/gojwe"
)

type Symwave struct{}

func (Symwave) Name() string {
	return "Symwave"
}

func (Symwave) Is(keySector []byte) bool {
	// note: stored little endian despite being a big endian system
	return keySector[3] == 'S' &&
		keySector[2] == 'Y' &&
		keySector[1] == 'M' &&
		keySector[0] == 'W'
}

func (Symwave) NeedsKEK() bool {
	return false
}

// The DEK is stored as two separately-wrapped halves.
// The KEK is only stored as one.
type keySector struct {
	Magic		[4]byte
	Unknown		[0xC]byte
	WrappedDEK1	[0x28]byte
	WrappedDEK2	[0x28]byte
	WrappedKEK	[0x28]byte
}

// This is hardcoded into the Symwave firmware.
var kekWrappingKey = []byte{
	0x29, 0xA2, 0x60, 0x7A,
	0xEA, 0x0B, 0x64, 0xAB,
	0x7B, 0xB3, 0xB9, 0xAB,
	0xA5, 0x69, 0x8B, 0x40,
	0x2E, 0x47, 0x93, 0xA6,
	0x81, 0x45, 0xC9, 0xCC,
	0x79, 0x94, 0x6A, 0x01,
	0x84, 0x0B, 0x34, 0xFE,
}

func (Symwave) ExtractDEK(keySector []byte, kek []byte) (dek []byte, err error) {
	var ks keySector

	r := bytes.NewReader(keySector)
	// Again, stored as little endian for some reason; this is a 68000 system so it should be big endian...
	err = binary.Read(r, binary.LittleEndian, &ks)
	if err != nil {
		return nil, err
	}

	// And again with the endianness stuff...
	wrapped := ks.WrappedKEK[:]
	SwapLongs(wrapped)
	kek, err = gojwe.AesKeyUnwrap(kekWrappingKey, wrapped)
	if err != nil {
		return nil, err
	}

	wrapped = ks.WrappedDEK1[:]
	SwapLongs(wrapped)
	dek1, err := gojwe.AesKeyUnwrap(kek, wrapped)
	if err != nil {
		return nil, err
	}

	wrapped = ks.WrappedDEK2[:]
	SwapLongs(wrapped)
	dek2, err := gojwe.AesKeyUnwrap(kek, wrapped)
	if err != nil {
		return nil, err
	}

	_ = dek2
	// And finally we just need one last endian correction...
	SwapLongs(dek1)
	return dek1, nil
}

func (Symwave) Decrypt(c cipher.Block, b []byte) {
	for i := 0; i < len(b); i += 16 {
		block := b[i : i+16]
		// ...and we can just use block as-is!
		c.Decrypt(block, block)
	}
}

func init() {
	bridge.Bridges = append(bridge.Bridges, Symwave{})
}