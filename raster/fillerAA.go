// Copyright 2011 The draw2d Authors. All rights reserved.
// created: 27/05/2011 by Laurent Le Goff
package raster

import (
	"image"
	"image/color"
	"unsafe"
)

const (
	SUBPIXEL_SHIFT         = 3
	SUBPIXEL_COUNT         = 1 << SUBPIXEL_SHIFT
	SUBPIXEL_FULL_COVERAGE = 0xff
)

var SUBPIXEL_OFFSETS = SUBPIXEL_OFFSETS_SAMPLE_8_FIXED

type SUBPIXEL_DATA uint8
type NON_ZERO_MASK_DATA_UNIT uint8

type Rasterizer8BitsSample struct {
	MaskBuffer    []SUBPIXEL_DATA
	WindingBuffer []NON_ZERO_MASK_DATA_UNIT

	Width           int
	BufferWidth     int
	Height          int
	ClipBound       [4]float64
	RemappingMatrix [6]float64
}

/*  width and height define the maximum output size for the filler.
 *  The filler will output to larger bitmaps as well, but the output will
 *  be cropped.
 */
func NewRasterizer8BitsSample(width, height int) *Rasterizer8BitsSample {
	var r Rasterizer8BitsSample
	// Scale the coordinates by SUBPIXEL_COUNT in vertical direction
	// The sampling point for the sub-pixel is at the top right corner. This
	// adjustment moves it to the pixel center.
	r.RemappingMatrix = [6]float64{1, 0, 0, SUBPIXEL_COUNT, 0.5 / SUBPIXEL_COUNT, -0.5 * SUBPIXEL_COUNT}
	r.Width = width
	r.Height = height
	// The buffer used for filling needs to be one pixel wider than the bitmap.
	// This is because the end flag that turns the fill of is the first pixel
	// after the actually drawn edge.
	r.BufferWidth = width + 1

	//r.MaskBuffer = make([]SUBPIXEL_DATA, r.BufferWidth*height)
	//r.WindingBuffer = make([]NON_ZERO_MASK_DATA_UNIT, r.BufferWidth*height*SUBPIXEL_COUNT)
	r.ClipBound = clip(0, 0, width, height, SUBPIXEL_COUNT)
	return &r
}

func clip(x, y, width, height, scale int) [4]float64 {
	var clipBound [4]float64

	offset := 0.99 / float64(scale)

	clipBound[0] = float64(x) + offset
	clipBound[2] = float64(x+width) - offset

	clipBound[1] = float64(y * scale)
	clipBound[3] = float64((y + height) * scale)
	return clipBound
}

func union(r1, r2 [4]float64) [4]float64 {
	if r1[0] > r2[0] {
		r1[0] = r2[0]
	}
	if r1[2] < r2[2] {
		r1[2] = r2[2]
	}
	if r1[1] > r2[1] {
		r1[1] = r2[1]
	}
	if r1[3] < r2[3] {
		r1[3] = r2[3]
	}
	return r1
}

func intersect(r1, r2 [4]float64) [4]float64 {
	if r1[0] < r2[0] {
		r1[0] = r2[0]
	}
	if r1[2] > r2[2] {
		r1[2] = r2[2]
	}
	if r1[1] < r2[1] {
		r1[1] = r2[1]
	}
	if r1[3] > r2[3] {
		r1[3] = r2[3]
	}
	return r1
}

func (r *Rasterizer8BitsSample) RenderEvenOdd(img *image.RGBA, color color.Color, polygons []Polygon, tr [6]float64) {
	// memset 0 the mask buffer
	r.MaskBuffer = make([]SUBPIXEL_DATA, r.BufferWidth*r.Height)

	// inline matrix multiplication
	transform := [6]float64{
		tr[0]*r.RemappingMatrix[0] + tr[1]*r.RemappingMatrix[2],
		tr[1]*r.RemappingMatrix[3] + tr[0]*r.RemappingMatrix[1],
		tr[2]*r.RemappingMatrix[0] + tr[3]*r.RemappingMatrix[2],
		tr[3]*r.RemappingMatrix[3] + tr[2]*r.RemappingMatrix[1],
		tr[4]*r.RemappingMatrix[0] + tr[5]*r.RemappingMatrix[2] + r.RemappingMatrix[4],
		tr[5]*r.RemappingMatrix[3] + tr[4]*r.RemappingMatrix[1] + r.RemappingMatrix[5],
	}

	clipRect := clip(img.Bounds().Min.X, img.Bounds().Min.Y, img.Bounds().Dx(), img.Bounds().Dy(), SUBPIXEL_COUNT)
	clipRect = intersect(clipRect, r.ClipBound)
	var edges [32]PolygonEdge
	bound := [4]float64{polygons[0][0], polygons[0][1], polygons[0][0], polygons[0][1]}
	for _, polygon := range polygons {
		p := 0
		l := len(polygon) / 2
		for p < l {
			edgeCount, subbound := polygon.getEdges(p, 16, edges[:], transform, clipRect)
			bound = union(bound, subbound)
			for k := 0; k < edgeCount; k++ {
				r.addEvenOddEdge(&edges[k])
			}
			p += 16
		}
	}
	clipRect = intersect(clipRect, bound)
	r.fillEvenOdd(img, color, clipRect)
}

//! Adds an edge to be used with even-odd fill.
func (r *Rasterizer8BitsSample) addEvenOddEdge(edge *PolygonEdge) {
	x := Fix(edge.X * FIXED_FLOAT_COEF)
	slope := Fix(edge.Slope * FIXED_FLOAT_COEF)
	slopeFix := Fix(0)
	if edge.LastLine-edge.FirstLine >= SLOPE_FIX_STEP {
		slopeFix = Fix(edge.Slope*SLOPE_FIX_STEP*FIXED_FLOAT_COEF) - (slope << SLOPE_FIX_SHIFT)
	}

	var mask SUBPIXEL_DATA
	var ySub uint32
	var xp, yLine int
	for y := edge.FirstLine; y <= edge.LastLine; y++ {
		ySub = uint32(y & (SUBPIXEL_COUNT - 1))
		xp = int((x + SUBPIXEL_OFFSETS[ySub]) >> FIXED_SHIFT)
		mask = SUBPIXEL_DATA(1 << ySub)
		yLine = y >> SUBPIXEL_SHIFT
		r.MaskBuffer[yLine*r.BufferWidth+xp] ^= mask
		x += slope
		if y&SLOPE_FIX_MASK == 0 {
			x += slopeFix
		}
	}
}

func convert(c color.Color) color.RGBA {
	if rgba, ok := c.(color.RGBA); ok {
		return rgba
	}
	r, g, b, a := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

// Renders the mask to the canvas with even-odd fill.
func (r *Rasterizer8BitsSample) fillEvenOdd(img *image.RGBA, c color.Color, clipBound [4]float64) {
	var x, y uint32

	minX := uint32(clipBound[0])
	maxX := uint32(clipBound[2])

	minY := uint32(clipBound[1]) >> SUBPIXEL_SHIFT
	maxY := uint32(clipBound[3]) >> SUBPIXEL_SHIFT

	rgba := convert(c)
	pixColor := *(*uint32)(unsafe.Pointer(&rgba))

	cs1 := pixColor & 0xff00ff
	cs2 := pixColor >> 8 & 0xff00ff

	stride := uint32(img.Stride)
	var mask SUBPIXEL_DATA
	maskY := minY * uint32(r.BufferWidth)
	minY *= stride
	maxY *= stride
	var tp []uint8
	var pixelx uint32
	for y = minY; y < maxY; y += stride {
		tp = img.Pix[y:]
		mask = 0

		//i0 := (s.Y-r.Image.Rect.Min.Y)*r.Image.Stride + (s.X0-r.Image.Rect.Min.X)*4
		//i1 := i0 + (s.X1-s.X0)*4
		pixelx = minX * 4
		for x = minX; x <= maxX; x++ {
			mask ^= r.MaskBuffer[maskY+x]
			// 8bits
			alpha := uint32(coverageTable[mask])
			// 16bits
			//alpha := uint32(coverageTable[mask & 0xff] + coverageTable[(mask >> 8) & 0xff])
			// 32bits
			//alpha := uint32(coverageTable[mask & 0xff] + coverageTable[(mask >> 8) & 0xff] + coverageTable[(mask >> 16) & 0xff] + coverageTable[(mask >> 24) & 0xff])

			// alpha is in range of 0 to SUBPIXEL_COUNT
			p := (*uint32)(unsafe.Pointer(&tp[pixelx]))
			if alpha == SUBPIXEL_FULL_COVERAGE {
				*p = pixColor
			} else if alpha != 0 {
				invAlpha := SUBPIXEL_COUNT - alpha
				ct1 := *p & 0xff00ff * invAlpha
				ct2 := *p >> 8 & 0xff00ff * invAlpha

				ct1 = (ct1 + cs1*alpha) >> SUBPIXEL_SHIFT & 0xff00ff
				ct2 = (ct2 + cs2*alpha) << (8 - SUBPIXEL_SHIFT) & 0xff00ff00

				*p = ct1 + ct2
			}
			pixelx += 4
		}
		maskY += uint32(r.BufferWidth)
	}
}

/*
 * Renders the polygon with non-zero winding fill.
 *  param aTarget the target bitmap.
 *  param aPolygon the polygon to render.
 *  param aColor the color to be used for rendering.
 *  param aTransformation the transformation matrix.
 */
func (r *Rasterizer8BitsSample) RenderNonZeroWinding(img *image.RGBA, color color.Color, polygons []Polygon, tr [6]float64) {

	r.MaskBuffer = make([]SUBPIXEL_DATA, r.BufferWidth*r.Height)
	r.WindingBuffer = make([]NON_ZERO_MASK_DATA_UNIT, r.BufferWidth*r.Height*SUBPIXEL_COUNT)

	// inline matrix multiplication
	transform := [6]float64{
		tr[0]*r.RemappingMatrix[0] + tr[1]*r.RemappingMatrix[2],
		tr[1]*r.RemappingMatrix[3] + tr[0]*r.RemappingMatrix[1],
		tr[2]*r.RemappingMatrix[0] + tr[3]*r.RemappingMatrix[2],
		tr[3]*r.RemappingMatrix[3] + tr[2]*r.RemappingMatrix[1],
		tr[4]*r.RemappingMatrix[0] + tr[5]*r.RemappingMatrix[2] + r.RemappingMatrix[4],
		tr[5]*r.RemappingMatrix[3] + tr[4]*r.RemappingMatrix[1] + r.RemappingMatrix[5],
	}

	clipRect := clip(img.Bounds().Min.X, img.Bounds().Min.Y, img.Bounds().Dx(), img.Bounds().Dy(), SUBPIXEL_COUNT)
	clipRect = intersect(clipRect, r.ClipBound)

	bound := [4]float64{polygons[0][0], polygons[0][1], polygons[0][0], polygons[0][1]}
	var edges [32]PolygonEdge
	for _, polygon := range polygons {
		p := 0
		l := len(polygon) / 2

		for p < l {
			edgeCount, subbound := polygon.getEdges(p, 16, edges[:], transform, clipRect)
			bound = union(bound, subbound)

			for k := 0; k < edgeCount; k++ {
				r.addNonZeroEdge(&edges[k])
			}
			p += 16
		}
	}
	clipRect = intersect(clipRect, bound)
	r.fillNonZero(img, color, clipRect)
}

//! Adds an edge to be used with non-zero winding fill.
func (r *Rasterizer8BitsSample) addNonZeroEdge(edge *PolygonEdge) {
	x := Fix(edge.X * FIXED_FLOAT_COEF)
	slope := Fix(edge.Slope * FIXED_FLOAT_COEF)
	slopeFix := Fix(0)
	if edge.LastLine-edge.FirstLine >= SLOPE_FIX_STEP {
		slopeFix = Fix(edge.Slope*SLOPE_FIX_STEP*FIXED_FLOAT_COEF) - slope<<SLOPE_FIX_SHIFT
	}
	var mask SUBPIXEL_DATA
	var ySub uint32
	var xp, yLine int
	winding := NON_ZERO_MASK_DATA_UNIT(edge.Winding)
	for y := edge.FirstLine; y <= edge.LastLine; y++ {
		ySub = uint32(y & (SUBPIXEL_COUNT - 1))
		xp = int((x + SUBPIXEL_OFFSETS[ySub]) >> FIXED_SHIFT)
		mask = SUBPIXEL_DATA(1 << ySub)
		yLine = y >> SUBPIXEL_SHIFT
		r.MaskBuffer[yLine*r.BufferWidth+xp] |= mask
		r.WindingBuffer[(yLine*r.BufferWidth+xp)*SUBPIXEL_COUNT+int(ySub)] += winding
		x += slope
		if y&SLOPE_FIX_MASK == 0 {
			x += slopeFix
		}
	}
}

//! Renders the mask to the canvas with non-zero winding fill.
func (r *Rasterizer8BitsSample) fillNonZero(img *image.RGBA, c color.Color, clipBound [4]float64) {
	var x, y uint32

	minX := uint32(clipBound[0])
	maxX := uint32(clipBound[2])

	minY := uint32(clipBound[1]) >> SUBPIXEL_SHIFT
	maxY := uint32(clipBound[3]) >> SUBPIXEL_SHIFT

	rgba := convert(c)
	pixColor := *(*uint32)(unsafe.Pointer(&rgba))
	cs1 := pixColor & 0xff00ff
	cs2 := pixColor >> 8 & 0xff00ff

	stride := uint32(img.Stride)
	var mask SUBPIXEL_DATA
	var n uint32
	var values [SUBPIXEL_COUNT]NON_ZERO_MASK_DATA_UNIT

	maskY := minY * uint32(r.BufferWidth)
	minY *= stride
	maxY *= stride
	var pixelx uint32
	for y = minY; y < maxY; y += stride {
		tp := img.Pix[y:]
		mask = 0
		pixelx = minX * 4
		for x = minX; x <= maxX; x++ {
			temp := r.MaskBuffer[maskY+x]
			if temp != 0 {
				var bit SUBPIXEL_DATA = 1
				for n = 0; n < SUBPIXEL_COUNT; n++ {
					if temp&bit != 0 {
						t := values[n]
						values[n] += r.WindingBuffer[(maskY+x)*SUBPIXEL_COUNT+n]
						if (t == 0 || values[n] == 0) && t != values[n] {
							mask ^= bit
						}
					}
					bit <<= 1
				}
			}

			// 8bits
			alpha := uint32(coverageTable[mask])
			// 16bits
			//alpha := uint32(coverageTable[mask & 0xff] + coverageTable[(mask >> 8) & 0xff])
			// 32bits
			//alpha := uint32(coverageTable[mask & 0xff] + coverageTable[(mask >> 8) & 0xff] + coverageTable[(mask >> 16) & 0xff] + coverageTable[(mask >> 24) & 0xff])

			p := (*uint32)(unsafe.Pointer(&tp[pixelx]))
			if alpha == SUBPIXEL_FULL_COVERAGE {
				*p = pixColor
			} else if alpha != 0 {
				// alpha is in range of 0 to SUBPIXEL_COUNT
				invAlpha := uint32(SUBPIXEL_COUNT) - alpha

				ct1 := *p & 0xff00ff * invAlpha
				ct2 := *p >> 8 & 0xff00ff * invAlpha

				ct1 = (ct1 + cs1*alpha) >> SUBPIXEL_SHIFT & 0xff00ff
				ct2 = (ct2 + cs2*alpha) << (8 - SUBPIXEL_SHIFT) & 0xff00ff00

				*p = ct1 + ct2
			}
			pixelx += 4
		}
		maskY += uint32(r.BufferWidth)
	}
}
