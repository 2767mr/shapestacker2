package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
)

type fatcatx struct {
	shape  [8]byte
	_      byte
	op     [2]byte
	_      byte
	input1 [8]byte
	_      byte
	input2 [8]byte
	_      byte
	cost   [2]byte
	_      byte
}

func readFromFatcatX() []Shape {
	// For all files in T:\\tmp\\ShapezTools\\db\\

	var result []Shape
	names, _ := os.ReadDir("T:\\tmp\\ShapezTools\\db\\")
	for _, name := range names {
		result = append(result, readFromFatcatXFile("T:\\tmp\\ShapezTools\\db\\"+name.Name())...)
	}
	return result
}

func readFromFatcatXFile(path string) []Shape {
	var result []Shape

	file, _ := os.Open(path)
	defer file.Close()

	var buffer bytes.Buffer
	io.Copy(&buffer, file)

	bytes := buffer.Bytes()

	mask1 := Shape(0x1111_1111)
	mask2 := Shape(0x8888_8888)
	mask3 := mask1 | mask2

	mask4 := Shape(0x2222_2222)
	mask5 := Shape(0x4444_4444)
	mask6 := mask4 | mask5

	var rawShape [4]byte
	for i := 0; i < len(bytes); i += 33 {
		for i := range rawShape {
			rawShape[i] = 0
		}
		hex.Decode(rawShape[:], bytes[i:i+8])
		s := Shape(binary.BigEndian.Uint32(rawShape[:]))

		s = s&^mask3 | (s&mask1)<<3 | (s&mask2)>>3
		s = s&^mask6 | (s&mask4)<<1 | (s&mask5)>>1

		result = append(result, s)
	}

	return result
}
