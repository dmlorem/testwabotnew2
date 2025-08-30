package media

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"math"

	"golang.org/x/image/draw"
)

func ResizeImg(data []byte, maxWidth int, maxHeight int) ([]byte, error) {
	if maxWidth <= 0 || maxHeight <= 0 {
		return nil, fmt.Errorf("maxWidth e maxHeight must be positive")
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("Error decoding image: %w", err)
	}

	originalWidth := img.Bounds().Dx()
	originalHeight := img.Bounds().Dy()

	if originalWidth <= maxWidth && originalHeight <= maxHeight {
		return data, nil
	}

	widthScale := float64(maxWidth) / float64(originalWidth)
	heightScale := float64(maxHeight) / float64(originalHeight)
	scale := math.Min(widthScale, heightScale)

	newWidth := int(float64(originalWidth) * scale)
	newHeight := int(float64(originalHeight) * scale)

	resizedImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.BiLinear.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, fmt.Errorf("Error decoding image: %w", err)
	}

	return buf.Bytes(), nil
}
