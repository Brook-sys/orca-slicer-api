package slicer

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
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
	hasThumbnail, err := gcodeHasThumbnail(gcodePath)
	if err != nil {
		return err
	}
	if hasThumbnail {
		return nil
	}
	img, err := renderNeptune4Thumbnail(modelData)
	if err != nil {
		return err
	}
	block, err := pngThumbnailBlock(img)
	if err != nil {
		return err
	}

	source, err := os.Open(gcodePath)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(gcodePath), ".thumbnail-*")
	if err != nil {
		_ = source.Close()
		return err
	}
	temporaryPath := temporary.Name()
	complete := false
	defer func() {
		_ = source.Close()
		_ = temporary.Close()
		if !complete {
			_ = os.Remove(temporaryPath)
		}
	}()
	if _, err := temporary.Write(block); err != nil {
		return err
	}
	if _, err := io.Copy(temporary, source); err != nil {
		return err
	}
	if err := temporary.Chmod(0o644); err != nil {
		return err
	}
	if err := source.Close(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, gcodePath); err != nil {
		return err
	}
	complete = true
	return nil
}

func gcodeHasThumbnail(gcodePath string) (bool, error) {
	file, err := os.Open(gcodePath)
	if err != nil {
		return false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxToolpathLineBytes)
	for scanner.Scan() {
		if bytes.Contains(scanner.Bytes(), []byte("; thumbnail begin")) {
			return true, nil
		}
	}
	return false, scanner.Err()
}

func renderNeptune4Thumbnail(modelData []byte) (*image.RGBA, error) {
	triangles, err := parseSTLTriangles(modelData)
	if err != nil {
		return nil, fmt.Errorf("thumbnail model parse: %w", err)
	}
	return renderThumbnail(triangles, 160, color.RGBA{R: 0, G: 0, B: 0, A: 0}), nil
}

func neptune4ThumbnailBase64(modelData []byte) (string, error) {
	img, err := renderNeptune4Thumbnail(modelData)
	if err != nil {
		return "", err
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pngData.Bytes()), nil
}

func pngThumbnailBlock(img image.Image) ([]byte, error) {
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(pngData.Bytes())
	var block strings.Builder
	block.WriteString("; THUMBNAIL_BLOCK_START\n\n;\n")
	block.WriteString(fmt.Sprintf("; thumbnail begin 160x160 %d\n", len(encoded)))
	for len(encoded) > 78 {
		block.WriteString("; ")
		block.WriteString(encoded[:78])
		block.WriteString("\n")
		encoded = encoded[78:]
	}
	if encoded != "" {
		block.WriteString("; ")
		block.WriteString(encoded)
		block.WriteString("\n")
	}
	block.WriteString("; thumbnail end\n; THUMBNAIL_BLOCK_END\n\n")
	return []byte(block.String()), nil
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
	// Neptune 4: teal model (#22948a) with shading
	baseR, baseG, baseB := uint8(0x22), uint8(0x94), uint8(0x8a)
	for _, tri := range projected {
		brightness := float64(tri.shade) / 255.0
		r := uint8(float64(baseR) * brightness)
		g := uint8(float64(baseG) * brightness)
		b := uint8(float64(baseB) * brightness)
		fillTriangle(img, tri.points, color.RGBA{R: r, G: g, B: b, A: 0xff})
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
