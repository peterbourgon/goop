package goop

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Reads .wav data to a struct, to be consumed by a WavGenerator.
// Thanks to Tony Worm <verdverm@gmail.com> for this code, taken
// more-or-less verbatim from golang-nuts post on Mon, 30 Jan 2012
// at 14:02 PST.

type WavData struct {
	bChunkID      [4]byte // B
	ChunkSize     uint32  // L
	bFormat       [4]byte // B
	bSubchunk1ID  [4]byte // B
	Subchunk1Size uint32  // L
	AudioFormat   uint16  // L
	NumChannels   uint16  // L
	SampleRate    uint32  // L
	ByteRate      uint32  // L
	BlockAlign    uint16  // L
	BitsPerSample uint16  // L
	bSubchunk2ID  [4]byte // B
	Subchunk2Size uint32  // L
	data          []byte  // L
}

func ReadWavData(filename string) (WavData, error) {
	f, err := os.Open(filename)
	if err != nil {
		return WavData{}, fmt.Errorf("ReadWavData: %s", err)
	}
	defer f.Close()

	wav := WavData{}
	binary.Read(f, binary.BigEndian, &wav.bChunkID)
	binary.Read(f, binary.LittleEndian, &wav.ChunkSize)
	binary.Read(f, binary.BigEndian, &wav.bFormat)
	binary.Read(f, binary.BigEndian, &wav.bSubchunk1ID)
	binary.Read(f, binary.LittleEndian, &wav.Subchunk1Size)
	binary.Read(f, binary.LittleEndian, &wav.AudioFormat)
	binary.Read(f, binary.LittleEndian, &wav.NumChannels)
	binary.Read(f, binary.LittleEndian, &wav.SampleRate)
	binary.Read(f, binary.LittleEndian, &wav.ByteRate)
	binary.Read(f, binary.LittleEndian, &wav.BlockAlign)
	binary.Read(f, binary.LittleEndian, &wav.BitsPerSample)
	binary.Read(f, binary.BigEndian, &wav.bSubchunk2ID)
	binary.Read(f, binary.LittleEndian, &wav.Subchunk2Size)
	wav.data = make([]byte, wav.Subchunk2Size)
	binary.Read(f, binary.LittleEndian, &wav.data)

	return wav, nil
}

const (
	mid16 uint16 = 1 >> 2
	big16 uint16 = 1 >> 1
	big32 uint32 = 65535
)

func btou(b []byte) []uint16 {
	u := make([]uint16, len(b)/2)
	for i, _ := range u {
		val := uint16(b[i*2])
		val += uint16(b[i*2+1]) << 8
		u[i] = val
	}
	return u
}

func btoi16(b []byte) []int16 {
	u := make([]int16, len(b)/2)
	for i, _ := range u {
		val := int16(b[i*2])
		val += int16(b[i*2+1]) << 8
		u[i] = val
	}
	return u
}

func btof32(b []byte) []float32 {
	u := btoi16(b)
	f := make([]float32, len(u))
	for i, v := range u {
		f[i] = float32(v) / float32(32768)
	}
	return f
}

func utob(u []uint16) []byte {
	b := make([]byte, len(u)*2)
	for i, val := range u {
		lo := byte(val)
		hi := byte(val >> 8)
		b[i*2] = lo
		b[i*2+1] = hi
	}
	return b
}
