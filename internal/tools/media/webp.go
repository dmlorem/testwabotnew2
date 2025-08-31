package media

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"slices"
)

type webpChunk struct {
	name [4]byte
	data []byte
}

func newWebpChunk(name [4]byte, data []byte) (*webpChunk, error) {
	for _, b := range name {
		if b < 32 || b > 126 {
			return nil, errors.New("invalid chunk name")
		}
	}
	return &webpChunk{name: name, data: data}, nil
}

func (c *webpChunk) toBytes() []byte {
	chunkSize := len(c.data)
	totalSize := 8 + chunkSize
	if chunkSize%2 != 0 {
		totalSize++
	}

	buf := make([]byte, 0, totalSize)
	buf = append(buf, c.name[:]...)
	size := make([]byte, 4)
	binary.LittleEndian.PutUint32(size, uint32(chunkSize))
	buf = append(buf, size...)
	buf = append(buf, c.data...)
	if chunkSize%2 != 0 {
		buf = append(buf, 0)
	}
	return buf
}

func parseWebp(data []byte) ([]*webpChunk, error) {
	if len(data) < 12 {
		return nil, errors.New("invalid file size")
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return nil, errors.New("not webp file")
	}

	expectedSize := binary.LittleEndian.Uint32(data[4:8]) + 8
	if uint32(len(data)) < expectedSize {
		return nil, errors.New("corrupted file")
	}

	var chunks []*webpChunk
	offset := 12

	for offset+8 <= len(data) {
		var name [4]byte
		copy(name[:], data[offset:offset+4])
		size := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		offset += 8

		if offset+int(size) > len(data) {
			return nil, errors.New("invalid chunk size")
		}

		chunkData := make([]byte, size)
		copy(chunkData, data[offset:offset+int(size)])
		offset += int(size)

		if size%2 != 0 && offset < len(data) {
			offset++
		}

		chunks = append(chunks, &webpChunk{name: name, data: chunkData})

		if offset >= int(expectedSize) {
			break
		}
	}

	return chunks, nil
}

func chunksToWebp(chunks []*webpChunk) []byte {
	vp8xChunk := []byte{
		'V', 'P', '8', 'X',
		0x0A, 0x00, 0x00, 0x00,
		0x00,
		0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
	}
	var vp8xFlags *byte
	var imageDimensions *[2]uint32

	filtered := make([]*webpChunk, 0, len(chunks))
	for _, c := range chunks {
		if string(c.name[:]) == "VP8X" {
			continue
		}
		switch string(c.name[:]) {
		case "VP8 ":
			width := uint32(binary.LittleEndian.Uint16(c.data[6:8]) & 0x3FFF)
			height := uint32(binary.LittleEndian.Uint16(c.data[8:10]) & 0x3FFF)
			imageDimensions = &[2]uint32{width, height}
		case "VP8L":
			bytes := binary.LittleEndian.Uint32(c.data[1:5])
			if (bytes>>28)&1 == 1 {
				setFlag(&vp8xFlags, 0b10000)
			}
			imageDimensions = &[2]uint32{(bytes & 0x3FFF) + 1, ((bytes >> 14) & 0x3FFF) + 1}
		case "ANMF":
			width := fromU24LE(c.data[7:10]) + 1
			height := fromU24LE(c.data[10:13]) + 1
			imageDimensions = &[2]uint32{width, height}
			setFlag(&vp8xFlags, 0b10)
		case "XMP ":
			setFlag(&vp8xFlags, 0b100)
		case "EXIF":
			setFlag(&vp8xFlags, 0b1000)
		case "ALPH":
			setFlag(&vp8xFlags, 0b10000)
		case "ICCP":
			setFlag(&vp8xFlags, 0b100000)
		}
		filtered = append(filtered, c)
	}

	data := []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P'}

	if vp8xFlags != nil && imageDimensions != nil {
		vp8xChunk[8] = *vp8xFlags
		copy(vp8xChunk[12:15], toU24LE(imageDimensions[0]-1))
		copy(vp8xChunk[15:18], toU24LE(imageDimensions[1]-1))
		data = append(data, vp8xChunk...)
	}

	for _, c := range filtered {
		data = append(data, c.toBytes()...)
	}

	size := uint32(len(data) - 8)
	binary.LittleEndian.PutUint32(data[4:8], size)
	return data
}

func toU24LE(n uint32) []byte {
	return []byte{byte(n), byte(n >> 8), byte(n >> 16)}
}
func fromU24LE(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16
}
func setFlag(flags **byte, mask byte) {
	if *flags == nil {
		tmp := byte(0)
		*flags = &tmp
	}
	**flags |= mask
}

func AddExifToWebp(webp []byte, title string, description string) ([]byte, error) {
	stickerJson := map[string]any{
		"sticker-pack-name":      title,
		"sticker-pack-publisher": description,
	}
	jsonBytes, err := json.Marshal(stickerJson)
	if err != nil {
		return nil, err
	}
	jsonBytes2 := []byte{
		0x49, 0x49, 0x2A, 0x00,
		0x08, 0x00, 0x00, 0x00,
		0x01, 0x00,
		0x41, 0x57, 0x07, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x16, 0x00, 0x00, 0x00,
	}
	jsonLength := uint32(len(jsonBytes))
	binary.LittleEndian.PutUint32(jsonBytes2[14:], jsonLength)
	data := append(jsonBytes2, jsonBytes...)
	exif, err := newWebpChunk([4]byte{'E', 'X', 'I', 'F'}, data)
	if err != nil {
		return nil, err
	}
	webpChunks, err := parseWebp(webp)
	if err != nil {
		return nil, err
	}

	webpChunks = slices.DeleteFunc(webpChunks, func(c *webpChunk) bool {
		return c.name == [4]byte{'E', 'X', 'I', 'F'}
	})

	webpChunks = append(webpChunks, exif)

	return chunksToWebp(webpChunks), nil
}
