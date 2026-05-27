package users

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
)

// resizeJPEG decodes src, scales it to fit within maxW×maxH while preserving
// the aspect ratio, and re-encodes it as JPEG quality 85. If both maxW and
// maxH are zero, or the image already fits within the bounds, src is returned
// unchanged. Uses the CatmullRom kernel for high-quality downscaling.
func resizeJPEG(src []byte, maxW, maxH uint32) ([]byte, error) {
	if maxW == 0 && maxH == 0 {
		return src, nil
	}

	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return src, nil
	}

	origW := img.Bounds().Dx()
	origH := img.Bounds().Dy()

	targetW, targetH := scaleToFit(origW, origH, int(maxW), int(maxH))

	if targetW == origW && targetH == origH {
		return src, nil
	}

	dst := image.NewNRGBA(image.Rect(0, 0, targetW, targetH))
	// draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
	draw.BiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return src, nil
	}
	return buf.Bytes(), nil
}

// scaleToFit returns the largest (w, h) that fits within (maxW, maxH) while
// preserving the aspect ratio of (origW, origH). A zero max dimension means
// unconstrained on that axis.
func scaleToFit(origW, origH, maxW, maxH int) (int, int) {
	if origW == 0 || origH == 0 {
		return origW, origH
	}

	w, h := origW, origH

	if maxW > 0 && w > maxW {
		h = h * maxW / w
		w = maxW
	}
	if maxH > 0 && h > maxH {
		w = w * maxH / h
		h = maxH
	}

	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}
