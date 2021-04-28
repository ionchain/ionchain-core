// Copyright 2016 The go-ionchain Authors
// This file is part of the go-ionchain library.
//
// The go-ionchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ionchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ionchain library. If not, see <http://www.gnu.org/licenses/>.
package base58

import (
	"errors"

	"golang.org/x/crypto/sha3"
)

// ErrChecksum indicates that the checksum of a check-encoded string does not verify against
// the checksum.
var ErrChecksum = errors.New("checksum error")

// ErrInvalidFormat indicates that the check-encoded string has an invalid format.
var ErrInvalidFormat = errors.New("invalid format: checksum bytes missing")

// checksum: first four bytes of SHA3-256
func checksum(input []byte) (cksum [4]byte) {
	sha := sha3.New256()
	sha.Write(input)
	hash := sha.Sum(nil)
	copy(cksum[:], hash[:4])
	return
}

// CheckEncode appends a four byte checksum.
func CheckEncode(input []byte) string {
	b := make([]byte, 0, len(input)+4)
	b = append(b, input[:]...)
	cksum := checksum(b)
	b = append(b, cksum[:]...)
	return Encode(b)
}

// CheckDecode decodes a string that was encoded with CheckEncode and verifies the checksum.
func CheckDecode(input string) (result []byte, err error) {
	decoded, err := Decode(input)
	if err != nil {
		return nil, err
	}
	if len(decoded) < 4 {
		return nil, ErrInvalidFormat
	}
	var cksum [4]byte
	copy(cksum[:], decoded[len(decoded)-4:])
	payload := decoded[:len(decoded)-4]
	if checksum(payload) != cksum {
		return nil, ErrChecksum
	}
	return payload, nil
}
