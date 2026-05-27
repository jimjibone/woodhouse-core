package users

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
)

// resizeJPEG decodes src as a JPEG (or PNG), scales it to fit within maxW×maxH
// while preserving the aspect ratio, and re-encodes it as JPEG. If maxW and
// maxH are both zero, or the image already fits within the bounds, src is
// returned unchanged.
func resizeJPEG(src []byte, maxW, maxH uint32) ([]byte, error) {
	if maxW == 0 && maxH == 0 {
		return src, nil
	}

	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return src, nil // return original if we can't decode
	}

	origW := img.Bounds().Dx()
	origH := img.Bounds().Dy()

	// Compute target dimensions, preserving aspect ratio.
	targetW, targetH := scaleToFit(origW, origH, int(maxW), int(maxH))

	// Nothing to do if target equals original.
	if targetW == origW && targetH == origH {
		return src, nil
	}

	dst := image.NewNRGBA(image.Rect(0, 0, targetW, targetH))
	drawBilinear(dst, img, targetW, targetH, origW, origH)

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

// drawBilinear scales src into dst using bilinear interpolation.
func drawBilinear(dst draw.Image, src image.Image, dstW, dstH, srcW, srcH int) {
	scaleX := float64(srcW) / float64(dstW)
	scaleY := float64(srcH) / float64(dstH)

	for dy := range dstH {
		sy := (float64(dy)+0.5)*scaleY - 0.5
		sy0 := int(sy)
		sy1 := sy0 + 1
		fy := sy - float64(sy0)
		if sy0 < 0 {
			sy0 = 0
		}
		if sy1 >= srcH {
			sy1 = srcH - 1
		}

		for dx := range dstW {
			sx := (float64(dx)+0.5)*scaleX - 0.5
			sx0 := int(sx)
			sx1 := sx0 + 1
			fx := sx - float64(sx0)
			if sx0 < 0 {
				sx0 = 0
			}
			if sx1 >= srcW {
				sx1 = srcW - 1
			}

			c00r, c00g, c00b, c00a := src.At(sx0, sy0).RGBA()
			c10r, c10g, c10b, c10a := src.At(sx1, sy0).RGBA()
			c01r, c01g, c01b, c01a := src.At(sx0, sy1).RGBA()
			c11r, c11g, c11b, c11a := src.At(sx1, sy1).RGBA()

			r := lerp2(c00r, c10r, c01r, c11r, fx, fy)
			g := lerp2(c00g, c10g, c01g, c11g, fx, fy)
			b := lerp2(c00b, c10b, c01b, c11b, fx, fy)
			a := lerp2(c00a, c10a, c01a, c11a, fx, fy)

			// RGBA() returns values in [0, 65535]; convert to [0, 255].
			dst.Set(dx, dy, &nrgbaColor{
				r: uint8(r >> 8),
				g: uint8(g >> 8),
				b: uint8(b >> 8),
				a: uint8(a >> 8),
			})
		}
	}
}

func lerp2(c00, c10, c01, c11 uint32, fx, fy float64) uint32 {
	top := float64(c00)*(1-fx) + float64(c10)*fx
	bot := float64(c01)*(1-fx) + float64(c11)*fx
	return uint32(top*(1-fy) + bot*fy)
}

// nrgbaColor is a minimal color.Color implementation.
type nrgbaColor struct{ r, g, b, a uint8 }

func (c *nrgbaColor) RGBA() (r, g, b, a uint32) {
	r = uint32(c.r)
	g = uint32(c.g)
	b = uint32(c.b)
	a = uint32(c.a)
	r |= r << 8
	g |= g << 8
	b |= b << 8
	a |= a << 8
	return
}
