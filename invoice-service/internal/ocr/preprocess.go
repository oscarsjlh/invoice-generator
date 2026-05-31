package ocr

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ValidateImageFile(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".heic", ".heif":
	default:
		return fmt.Errorf("unsupported image format %q, only JPEG, PNG, HEIC, and HEIF are accepted", ext)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open image: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if ext == ".heic" || ext == ".heif" {
		if ok, err := hasHEIFSignature(file); err != nil {
			return fmt.Errorf("cannot inspect HEIC/HEIF image: %w", err)
		} else if !ok {
			return fmt.Errorf("file content is not a valid HEIC/HEIF image")
		}
		return nil
	}

	_, format, err := image.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("cannot decode image: %w", err)
	}

	format = strings.ToLower(format)
	if format != "jpeg" && format != "png" {
		return fmt.Errorf("file content is not a valid JPEG or PNG image (detected %q)", format)
	}

	return nil
}

func hasHEIFSignature(file *os.File) (bool, error) {
	header := make([]byte, 32)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return false, err
	}
	if n < 12 || string(header[4:8]) != "ftyp" {
		return false, nil
	}
	brands := [][]byte{
		[]byte("heic"),
		[]byte("heix"),
		[]byte("hevc"),
		[]byte("hevx"),
		[]byte("heim"),
		[]byte("heis"),
		[]byte("hevm"),
		[]byte("hevs"),
		[]byte("mif1"),
		[]byte("msf1"),
	}
	for _, brand := range brands {
		if bytes.Contains(header[8:n], brand) {
			return true, nil
		}
	}
	return false, nil
}

func PreprocessImage(srcPath, dstPath string) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	maxDim := 2048
	width := bounds.Dx()
	height := bounds.Dy()

	alignMultiple := 32

	var newWidth, newHeight int
	if width > maxDim || height > maxDim {
		if width > height {
			newWidth = maxDim
			newHeight = height * maxDim / width
		} else {
			newHeight = maxDim
			newWidth = width * maxDim / height
		}
	} else {
		newWidth = width
		newHeight = height
	}

	newWidth = ((newWidth + alignMultiple - 1) / alignMultiple) * alignMultiple
	newHeight = ((newHeight + alignMultiple - 1) / alignMultiple) * alignMultiple

	var newImg image.Image
	if newWidth != width || newHeight != height {
		resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		for y := 0; y < newHeight; y++ {
			for x := 0; x < newWidth; x++ {
				srcX := x * width / newWidth
				srcY := y * height / newHeight
				resized.Set(x, y, img.At(srcX, srcY))
			}
		}
		newImg = resized
	} else {
		newImg = img
	}

	outFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer func() {
		_ = outFile.Close()
	}()

	return jpeg.Encode(outFile, newImg, &jpeg.Options{Quality: 85})
}

func NormalizeOCREntry(raw, fieldName string, knownCategories map[string]bool) OCRExtractedField {
	normalized := strings.TrimSpace(raw)
	confidence := 1.0
	needsReview := false

	if normalized == "" {
		confidence = 0.0
		needsReview = true
		return OCRExtractedField{
			Raw:         raw,
			Normalized:  "",
			Confidence:  confidence,
			NeedsReview: needsReview,
		}
	}

	if fieldName == "category" {
		if _, ok := knownCategories[normalized]; ok {
			return OCRExtractedField{
				Raw:         raw,
				Normalized:  normalized,
				Confidence:  0.95,
				NeedsReview: false,
			}
		}

		var candidates []string
		lower := strings.ToLower(normalized)
		for cat := range knownCategories {
			if strings.EqualFold(normalized, cat) {
				candidates = append(candidates, cat)
			} else if strings.Contains(strings.ToLower(cat), lower) || strings.Contains(lower, strings.ToLower(cat)) {
				candidates = append(candidates, cat)
			}
		}

		if len(candidates) == 1 {
			confidence = 0.80
			return OCRExtractedField{
				Raw:         raw,
				Normalized:  candidates[0],
				Candidates:  candidates,
				Confidence:  confidence,
				NeedsReview: false,
			}
		} else if len(candidates) > 1 {
			confidence = 0.60
			return OCRExtractedField{
				Raw:         raw,
				Normalized:  normalized,
				Candidates:  candidates,
				Confidence:  confidence,
				NeedsReview: true,
			}
		}

		confidence = 0.40
		needsReview = true
	}

	return OCRExtractedField{
		Raw:         raw,
		Normalized:  normalized,
		Confidence:  confidence,
		NeedsReview: needsReview,
	}
}
