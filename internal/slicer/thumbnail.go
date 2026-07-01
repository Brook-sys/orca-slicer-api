package slicer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type point3 struct {
	x float64
	y float64
	z float64
}

type triangle3 struct {
	a point3
	b point3
	c point3
}

type projectedTriangle struct {
	points [3]image.Point
	depth  float64
	shade  uint8
}

func addNeptune4ThumbnailsToGCode(gcodePath string, modelData []byte) error {
	data, err := os.ReadFile(gcodePath)
	if err != nil {
		return err
	}
	if bytes.Contains(data, []byte(";gimage:")) || bytes.Contains(data, []byte(";simage:")) {
		return nil
	}
	triangles, err := parseSTLTriangles(modelData)
	if err != nil {
		return fmt.Errorf("thumbnail model parse: %w", err)
	}
	large := renderThumbnail(triangles, 320, color.RGBA{R: 0x30, G: 0x39, B: 0x4f, A: 0xff})
	small := renderThumbnail(triangles, 160, color.RGBA{R: 0x30, G: 0x39, B: 0x4f, A: 0xff})
	largeEncoded := encodeColPic(large)
	smallEncoded := encodeColPic(small)
	if largeEncoded == "" || smallEncoded == "" {
		return errors.New("thumbnail COLPIC encode failed")
	}
	prefix := []byte(";gimage:" + largeEncoded + "\n\n;simage:" + smallEncoded + "\n\n")
	return os.WriteFile(gcodePath, append(prefix, data...), 0o644)
}

func parseSTLTriangles(data []byte) ([]triangle3, error) {
	if triangles, ok := parseBinarySTL(data); ok {
		return triangles, nil
	}
	return parseASCIISTL(data)
}

func parseBinarySTL(data []byte) ([]triangle3, bool) {
	if len(data) < 84 {
		return nil, false
	}
	count := binary.LittleEndian.Uint32(data[80:84])
	expected := 84 + int(count)*50
	if count == 0 || expected > len(data) {
		return nil, false
	}
	triangles := make([]triangle3, 0, count)
	offset := 84
	for i := 0; i < int(count); i++ {
		offset += 12
		triangles = append(triangles, triangle3{
			a: readSTLPoint(data[offset : offset+12]),
			b: readSTLPoint(data[offset+12 : offset+24]),
			c: readSTLPoint(data[offset+24 : offset+36]),
		})
		offset += 38
	}
	return triangles, true
}

func readSTLPoint(data []byte) point3 {
	return point3{
		x: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[0:4]))),
		y: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[4:8]))),
		z: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[8:12]))),
	}
}

func parseASCIISTL(data []byte) ([]triangle3, error) {
	fields := strings.Fields(string(data))
	vertices := make([]point3, 0)
	for i := 0; i+3 < len(fields); i++ {
		if fields[i] != "vertex" {
			continue
		}
		x, xerr := strconv.ParseFloat(fields[i+1], 64)
		y, yerr := strconv.ParseFloat(fields[i+2], 64)
		z, zerr := strconv.ParseFloat(fields[i+3], 64)
		if xerr != nil || yerr != nil || zerr != nil {
			continue
		}
		vertices = append(vertices, point3{x: x, y: y, z: z})
	}
	if len(vertices) < 3 {
		return nil, errors.New("no STL triangles found")
	}
	triangles := make([]triangle3, 0, len(vertices)/3)
	for i := 0; i+2 < len(vertices); i += 3 {
		triangles = append(triangles, triangle3{a: vertices[i], b: vertices[i+1], c: vertices[i+2]})
	}
	return triangles, nil
}

func renderThumbnail(triangles []triangle3, size int, background color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, background)
		}
	}
	if len(triangles) == 0 {
		return img
	}

	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, tri := range triangles {
		for _, p := range []point3{tri.a, tri.b, tri.c} {
			x := p.x - p.y
			y := (p.x+p.y)*0.5 - p.z
			minX = math.Min(minX, x)
			minY = math.Min(minY, y)
			maxX = math.Max(maxX, x)
			maxY = math.Max(maxY, y)
		}
	}
	scale := float64(size) * 0.78 / math.Max(maxX-minX, maxY-minY)
	if scale <= 0 || math.IsInf(scale, 0) || math.IsNaN(scale) {
		return img
	}
	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2

	projected := make([]projectedTriangle, 0, len(triangles))
	for _, tri := range triangles {
		points := [3]image.Point{}
		depth := 0.0
		for i, p := range []point3{tri.a, tri.b, tri.c} {
			x := p.x - p.y
			y := (p.x+p.y)*0.5 - p.z
			points[i] = image.Point{X: int(math.Round(float64(size)/2 + (x-centerX)*scale)), Y: int(math.Round(float64(size)/2 + (y-centerY)*scale))}
			depth += p.x + p.y + p.z
		}
		shade := triangleShade(tri)
		projected = append(projected, projectedTriangle{points: points, depth: depth / 3, shade: shade})
	}
	sort.Slice(projected, func(i, j int) bool { return projected[i].depth < projected[j].depth })
	for _, tri := range projected {
		fillTriangle(img, tri.points, color.RGBA{R: tri.shade, G: tri.shade, B: tri.shade, A: 0xff})
	}
	return img
}

func triangleShade(tri triangle3) uint8 {
	u := point3{x: tri.b.x - tri.a.x, y: tri.b.y - tri.a.y, z: tri.b.z - tri.a.z}
	v := point3{x: tri.c.x - tri.a.x, y: tri.c.y - tri.a.y, z: tri.c.z - tri.a.z}
	n := point3{x: u.y*v.z - u.z*v.y, y: u.z*v.x - u.x*v.z, z: u.x*v.y - u.y*v.x}
	length := math.Sqrt(n.x*n.x + n.y*n.y + n.z*n.z)
	if length == 0 {
		return 185
	}
	n.x, n.y, n.z = n.x/length, n.y/length, n.z/length
	light := point3{x: -0.35, y: -0.45, z: 0.82}
	dot := math.Abs(n.x*light.x + n.y*light.y + n.z*light.z)
	return uint8(115 + dot*110)
}

func fillTriangle(img *image.RGBA, points [3]image.Point, c color.RGBA) {
	minX := max(0, min(points[0].X, min(points[1].X, points[2].X)))
	maxX := min(img.Bounds().Dx()-1, max(points[0].X, max(points[1].X, points[2].X)))
	minY := max(0, min(points[0].Y, min(points[1].Y, points[2].Y)))
	maxY := min(img.Bounds().Dy()-1, max(points[0].Y, max(points[1].Y, points[2].Y)))
	area := edge(points[0], points[1], points[2])
	if area == 0 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			p := image.Point{X: x, Y: y}
			w0 := edge(points[1], points[2], p)
			w1 := edge(points[2], points[0], p)
			w2 := edge(points[0], points[1], p)
			if (w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func edge(a, b, c image.Point) int {
	return (c.X-a.X)*(b.Y-a.Y) - (c.Y-a.Y)*(b.X-a.X)
}

func encodeColPic(img *image.RGBA) string {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	colors := make([]uint16, width*height)
	index := len(colors) - 1
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			px := img.RGBAAt(width-col-1, row)
			r := uint16(px.R) >> 3
			g := uint16(px.G) >> 2
			b := uint16(px.B) >> 3
			if px.A == 0 {
				r = 46 >> 3
				g = 51 >> 2
				b = 72 >> 3
			}
			colors[index] = (r << 11) | (g << 5) | b
			index--
		}
	}
	encoded := colPicEncode(colors, width, height, 1024)
	return colPicEncodeString(encoded)
}

type colorHead struct {
	color uint16
	qty   int
}

func colPicEncode(colors []uint16, width int, height int, colorsMax int) []byte {
	if colorsMax > 1024 {
		colorsMax = 1024
	}
	palette := make([]colorHead, 0, colorsMax)
	positions := map[uint16]int{}
	for _, c := range colors {
		if pos, ok := positions[c]; ok {
			palette[pos].qty++
			continue
		}
		positions[c] = len(palette)
		palette = append(palette, colorHead{color: c, qty: 1})
	}
	sort.SliceStable(palette, func(i, j int) bool { return palette[i].qty > palette[j].qty })
	for len(palette) > colorsMax {
		last := palette[len(palette)-1]
		closest := 0
		closestDistance := math.MaxInt
		for i := 0; i < colorsMax; i++ {
			distance := colorDistance(last.color, palette[i].color)
			if distance < closestDistance {
				closestDistance = distance
				closest = i
			}
		}
		for i, c := range colors {
			if c == last.color {
				colors[i] = palette[closest].color
			}
		}
		palette = palette[:len(palette)-1]
	}
	paletteColors := make([]uint16, len(palette))
	paletteIndex := make(map[uint16]int, len(palette))
	for i, p := range palette {
		paletteColors[i] = p.color
		paletteIndex[p.color] = i
	}
	body := byte8bitEncode(colors, paletteIndex, len(colors))
	out := make([]byte, 32+len(paletteColors)*2+len(body))
	out[0] = 3
	binary.LittleEndian.PutUint32(out[8:12], uint32(width))
	binary.LittleEndian.PutUint32(out[12:16], uint32(height))
	binary.LittleEndian.PutUint32(out[16:20], 0x05DDC33C)
	binary.LittleEndian.PutUint32(out[20:24], uint32(len(paletteColors)*2))
	binary.LittleEndian.PutUint32(out[24:28], uint32(len(body)))
	paletteOffset := 32
	for i, c := range paletteColors {
		binary.LittleEndian.PutUint16(out[paletteOffset+i*2:paletteOffset+i*2+2], c)
	}
	copy(out[paletteOffset+len(paletteColors)*2:], body)
	return out
}

func colorDistance(a uint16, b uint16) int {
	ar, ag, ab := int((a>>11)&0x1f), int((a>>5)&0x3f), int(a&0x1f)
	br, bg, bb := int((b>>11)&0x1f), int((b>>5)&0x3f), int(b&0x1f)
	return abs(ar-br) + abs(ag-bg) + abs(ab-bb)
}

func byte8bitEncode(colors []uint16, palette map[uint16]int, dotsQty int) []byte {
	out := make([]byte, 0, dotsQty)
	src := 0
	lastSID := 0
	for dotsQty > 0 {
		dots := 1
		for i := 0; i < dotsQty-1; i++ {
			if colors[src+i] != colors[src+i+1] {
				break
			}
			dots++
			if dots == 255 {
				break
			}
		}
		paletteID := palette[colors[src]]
		tid := byte(paletteID % 32)
		sid := byte(paletteID / 32)
		if lastSID != int(sid) {
			out = append(out, (7<<5)+sid)
			lastSID = int(sid)
		}
		if dots <= 6 {
			out = append(out, byte(dots<<5)+tid)
		} else {
			out = append(out, tid, byte(dots))
		}
		src += dots
		dotsQty -= dots
	}
	return out
}

func colPicEncodeString(data []byte) string {
	padding := 3 - (len(data) % 3)
	for padding > 0 {
		data = append(data, 0)
		padding--
	}
	out := make([]byte, len(data)*4/3)
	outIndex := len(out)
	for hexIndex := len(data); hexIndex > 0; {
		hexIndex -= 3
		outIndex -= 4
		temp0 := data[hexIndex] >> 2
		temp1 := ((data[hexIndex] & 3) << 4) + (data[hexIndex+1] >> 4)
		temp2 := ((data[hexIndex+1] & 15) << 2) + (data[hexIndex+2] >> 6)
		temp3 := data[hexIndex+2] & 63
		out[outIndex] = colPicChar(temp0)
		out[outIndex+1] = colPicChar(temp1)
		out[outIndex+2] = colPicChar(temp2)
		out[outIndex+3] = colPicChar(temp3)
	}
	return string(out)
}

func colPicChar(value byte) byte {
	value += 48
	if value == '\\' {
		return '~'
	}
	return value
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
