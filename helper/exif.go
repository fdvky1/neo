package helper

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
)

type StickerMetadata struct {
	PackId    string   `json:"sticker-pack-id"`
	Name      string   `json:"sticker-pack-name"`
	Publisher string   `json:"sticker-pack-publisher"`
	Emojis    []string `json:"emojis,omitempty"`
}

func writeUIntLE(buffer []byte, value, offset, byteLength int64) {
	slice := make([]byte, byteLength)
	val := new(big.Int)
	val.SetUint64(uint64(value))
	valBytes := val.Bytes()

	tmp := make([]byte, len(valBytes))
	for i := range valBytes {
		tmp[i] = valBytes[len(valBytes)-1-i]
	}
	copy(slice, tmp)
	copy(buffer[offset:], slice)
}

func createMetadataBytes(packName, publisher string) []byte {
	exifObj := StickerMetadata{
		PackId:    "com.wa.bot.sticker",
		Name:      packName,
		Publisher: publisher,
		Emojis:    []string{"😀"},
	}

	exifJson, err := json.Marshal(exifObj)
	if err != nil {
		return nil
	}

	bit := []byte{0x49, 0x49, 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, 0x01, 0x00, 0x41, 0x57, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00}
	bit = append(bit, exifJson...)
	writeUIntLE(bit, int64(len(exifJson)), 14, 4)

	return bit
}

// InjectExif takes raw WebP bytes and constructs a new WebP RIFF byte slice containing the EXIF metadata
func InjectExif(webpData []byte, packName, publisher string) ([]byte, error) {
	if len(webpData) < 12 || string(webpData[0:4]) != "RIFF" || string(webpData[8:12]) != "WEBP" {
		return nil, fmt.Errorf("not a valid WEBP file")
	}

	var width, height uint32
	hasVP8X := false
	var vp8xFlags byte
	var vp8xBytes []byte

	var chunks [][]byte

	// Default defaults for VP8X canvas sizes
	width = 512
	height = 512

	offset := 12
	for offset < len(webpData) {
		if offset+8 > len(webpData) {
			break
		}
		chunkID := string(webpData[offset : offset+4])
		chunkSize := binary.LittleEndian.Uint32(webpData[offset+4 : offset+8])

		paddedSize := chunkSize
		if paddedSize%2 != 0 {
			paddedSize++
		}

		if offset+8+int(paddedSize) > len(webpData) {
			break
		}

		chunkData := webpData[offset+8 : offset+8+int(paddedSize)]

		switch chunkID {
		case "VP8X":
			hasVP8X = true
			vp8xFlags = chunkData[0]
			vp8xBytes = chunkData
			// extract dimensions
			wBytes := []byte{chunkData[4], chunkData[5], chunkData[6], 0}
			hBytes := []byte{chunkData[7], chunkData[8], chunkData[9], 0}
			width = binary.LittleEndian.Uint32(wBytes) + 1
			height = binary.LittleEndian.Uint32(hBytes) + 1
		case "VP8 ":
			if !hasVP8X {
				// extract 14 bit scale block
				w := uint32(chunkData[6]) | (uint32(chunkData[7]) << 8)
				width = w & 0x3FFF
				h := uint32(chunkData[8]) | (uint32(chunkData[9]) << 8)
				height = h & 0x3FFF
			}
			chunks = append(chunks, webpData[offset:offset+8+int(paddedSize)])
		case "VP8L":
			if !hasVP8X {
				// Parse bits 1-28 for w/h
				b0 := uint32(chunkData[1])
				b1 := uint32(chunkData[2])
				b2 := uint32(chunkData[3])
				b3 := uint32(chunkData[4])
				width = 1 + (((b1 & 0x3F) << 8) | b0)
				height = 1 + (((b3 & 0xF) << 10) | (b2 << 2) | ((b1 & 0xC0) >> 6))
			}
			chunks = append(chunks, webpData[offset:offset+8+int(paddedSize)])
		default:
			if chunkID != "EXIF" {
				chunks = append(chunks, webpData[offset:offset+8+int(paddedSize)])
			}
		}
		offset += 8 + int(paddedSize)
	}

	exifMeta := createMetadataBytes(packName, publisher)
	if exifMeta == nil {
		return nil, fmt.Errorf("failed to create exif byte payload")
	}

	// Turn on EXIF flag (bit 3) in VP8X
	if !hasVP8X {
		vp8xBytes = make([]byte, 10)
		vp8xFlags = 0x08 // EXIF flag is bit 3
		vp8xBytes[0] = vp8xFlags
		w := width - 1
		h := height - 1
		vp8xBytes[4] = byte(w)
		vp8xBytes[5] = byte(w >> 8)
		vp8xBytes[6] = byte(w >> 16)
		vp8xBytes[7] = byte(h)
		vp8xBytes[8] = byte(h >> 8)
		vp8xBytes[9] = byte(h >> 16)
	} else {
		vp8xBytes[0] = vp8xFlags | 0x08
	}

	var buf bytes.Buffer
	buf.WriteString("RIFF")
	buf.Write([]byte{0, 0, 0, 0}) // Total size placeholder
	buf.WriteString("WEBP")

	// Write VP8X
	buf.WriteString("VP8X")
	vp8xSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(vp8xSize, 10)
	buf.Write(vp8xSize)
	buf.Write(vp8xBytes)

	// Write existing chunks
	for _, chunk := range chunks {
		buf.Write(chunk)
	}

	// Write EXIF
	buf.WriteString("EXIF")
	exifSize := uint32(len(exifMeta))
	exifSizeSlice := make([]byte, 4)
	binary.LittleEndian.PutUint32(exifSizeSlice, exifSize)
	buf.Write(exifSizeSlice)
	buf.Write(exifMeta)
	if exifSize%2 != 0 {
		buf.WriteByte(0)
	}

	result := buf.Bytes()
	totalSize := uint32(len(result) - 8)
	binary.LittleEndian.PutUint32(result[4:8], totalSize)

	return result, nil
}