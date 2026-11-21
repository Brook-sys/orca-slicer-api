package slicer

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	maxToolpathSourceBytes = 32 << 20
	maxToolpathLineBytes   = 1 << 20
	maxToolpathLines       = 4_000_000
	maxToolpathLayers      = 512
	maxToolpathSegments    = 20_000
	maxToolpathArcSegments = 4_096
	maxToolpathArcWork     = 1_000_000
	toolpathArcToleranceMM = 0.05
)

type ToolpathPreview struct {
	Version          int             `json:"version"`
	PositionUnit     string          `json:"positionUnit"`
	ExtrusionUnit    string          `json:"extrusionUnit"`
	FeedrateUnit     string          `json:"feedrateUnit"`
	CoordinateSystem string          `json:"coordinateSystem"`
	SourceComplete   bool            `json:"sourceComplete"`
	Truncated        bool            `json:"truncated"`
	SourceBytes      int64           `json:"sourceBytes"`
	SourceLines      int             `json:"sourceLines"`
	TotalLayers      int             `json:"totalLayers"`
	TotalSegments    int             `json:"totalSegments"`
	ReturnedLayers   int             `json:"returnedLayers"`
	ReturnedSegments int             `json:"returnedSegments"`
	OmittedSegments  int             `json:"omittedSegments"`
	SupportSegments  int             `json:"supportSegments"`
	BrimSegments     int             `json:"brimSegments"`
	LayerSemantics   string          `json:"layerSemantics"`
	Warnings         []string        `json:"warnings"`
	Limits           ToolpathLimits  `json:"limits"`
	Bounds           *ToolpathBounds `json:"bounds,omitempty"`
	Layers           []ToolpathLayer `json:"layers"`
}

type ToolpathLimits struct {
	SourceBytes         int     `json:"sourceBytes"`
	LineBytes           int     `json:"lineBytes"`
	Lines               int     `json:"lines"`
	Layers              int     `json:"layers"`
	Segments            int     `json:"segments"`
	ArcSegmentsPerMove  int     `json:"arcSegmentsPerMove"`
	ArcSegmentsTotal    int     `json:"arcSegmentsTotal"`
	ArcChordToleranceMM float64 `json:"arcChordToleranceMm"`
}

type ToolpathBounds struct {
	Min [3]float64 `json:"min"`
	Max [3]float64 `json:"max"`
}

type ToolpathLayer struct {
	Index    int               `json:"index"`
	Z        float64           `json:"z"`
	Segments []ToolpathSegment `json:"segments"`
}

type ToolpathSegment struct {
	From      [3]float64 `json:"from"`
	To        [3]float64 `json:"to"`
	Extrusion float64    `json:"extrusion"`
	Feedrate  float64    `json:"feedrateMmSec,omitempty"`
	Feature   string     `json:"feature,omitempty"`
	Object    string     `json:"object,omitempty"`
	Tool      int        `json:"tool"`
}

type toolpathParser struct {
	ctx              context.Context
	preview          ToolpathPreview
	position         [3]float64
	origin           [3]float64
	extruderOrigin   float64
	extruderPosition float64
	feedrate         float64
	unitsScale       float64
	globalRelative   bool
	extruderRelative bool
	currentTool      int
	currentFeature   string
	currentObject    string
	wiping           bool
	sawLayerMarker   bool
	pendingLayerZ    bool
	currentLayer     int
	retainedLayer    int
	fallbackZ        float64
	retainedSegments int
	arcWork          int
	thumbnailActive  bool
	thumbnailLeft    int
	thumbnailPadding int
}

var objectLabelPattern = regexp.MustCompile(`^(.*) id:([0-9]+) copy ([0-9]+)$`)

// These commands do not encode additional XYZ/E motion in the G-code stream.
// Unknown M commands fail closed: some can home, park, or select tools.
var nonMotionMCommands = map[string]bool{
	"M18": true, "M73": true, "M84": true, "M104": true, "M106": true,
	"M107": true, "M109": true, "M117": true, "M140": true, "M190": true,
	"M201": true, "M203": true, "M204": true, "M205": true, "M220": true,
	"M221": true, "M300": true, "M400": true, "M486": true, "M566": true,
	"M572": true, "M624": true, "M625": true, "M900": true,
}

func extractToolpathPreview(ctx context.Context, path string) (ToolpathPreview, error) {
	file, err := os.Open(path)
	if err != nil {
		return ToolpathPreview{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return ToolpathPreview{}, err
	}
	if info.Size() > maxToolpathSourceBytes {
		return ToolpathPreview{}, fmt.Errorf("G-code exceeds the %d-byte preview limit", int64(maxToolpathSourceBytes))
	}

	parser := toolpathParser{
		ctx:           ctx,
		unitsScale:    1,
		currentLayer:  -1,
		retainedLayer: -1,
		fallbackZ:     math.NaN(),
	}
	parser.preview = ToolpathPreview{
		Version:          1,
		PositionUnit:     "mm",
		ExtrusionUnit:    "mm",
		FeedrateUnit:     "mm/s",
		CoordinateSystem: "gcode",
		SourceBytes:      info.Size(),
		LayerSemantics:   "z_fallback",
		Warnings:         make([]string, 0),
		Layers:           make([]ToolpathLayer, 0),
		Limits: ToolpathLimits{
			SourceBytes:         maxToolpathSourceBytes,
			LineBytes:           maxToolpathLineBytes,
			Lines:               maxToolpathLines,
			Layers:              maxToolpathLayers,
			Segments:            maxToolpathSegments,
			ArcSegmentsPerMove:  maxToolpathArcSegments,
			ArcSegmentsTotal:    maxToolpathArcWork,
			ArcChordToleranceMM: toolpathArcToleranceMM,
		},
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxToolpathLineBytes)
	for scanner.Scan() {
		parser.preview.SourceLines++
		if parser.preview.SourceLines > maxToolpathLines {
			return ToolpathPreview{}, fmt.Errorf("G-code exceeds the %d-line preview limit", maxToolpathLines)
		}
		if parser.preview.SourceLines%1024 == 0 {
			select {
			case <-ctx.Done():
				return ToolpathPreview{}, ctx.Err()
			default:
			}
		}
		if err := parser.parseLine(strings.TrimSuffix(scanner.Text(), "\r")); err != nil {
			return ToolpathPreview{}, fmt.Errorf("line %d: %w", parser.preview.SourceLines, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return ToolpathPreview{}, fmt.Errorf("read G-code (maximum line length %d bytes): %w", maxToolpathLineBytes, err)
	}
	select {
	case <-ctx.Done():
		return ToolpathPreview{}, ctx.Err()
	default:
	}

	if parser.thumbnailActive {
		return ToolpathPreview{}, fmt.Errorf("unclosed thumbnail block")
	}

	parser.preview.SourceComplete = true
	parser.preview.ReturnedLayers = len(parser.preview.Layers)
	parser.preview.ReturnedSegments = 0
	for _, layer := range parser.preview.Layers {
		parser.preview.ReturnedSegments += len(layer.Segments)
	}
	parser.preview.OmittedSegments = parser.preview.TotalSegments - parser.preview.ReturnedSegments
	parser.preview.Truncated = parser.preview.TotalLayers > len(parser.preview.Layers) || parser.preview.OmittedSegments > 0
	return parser.preview, nil
}

func (p *toolpathParser) parseLine(raw string) error {
	line := strings.TrimSpace(raw)
	if p.thumbnailActive {
		// Only full-line, base64 comments may be skipped. In particular, never
		// hide executable motion or nested/malformed markers inside a thumbnail.
		if !strings.HasPrefix(line, ";") {
			return fmt.Errorf("expected thumbnail payload comment or end marker")
		}
		return p.parseThumbnailPayload(strings.TrimSpace(line[1:]))
	}
	if line == "" {
		return nil
	}
	if strings.HasPrefix(line, ";") {
		return p.parseComment(strings.TrimSpace(strings.TrimPrefix(line, ";")))
	}
	if index := strings.IndexByte(line, ';'); index >= 0 {
		line = strings.TrimSpace(line[:index])
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	// Accept standard line numbers and checksums without treating them as axes.
	if len(fields[0]) > 1 && fields[0][0] == 'N' {
		if _, err := strconv.Atoi(fields[0][1:]); err == nil {
			fields = fields[1:]
		}
	}
	if len(fields) == 0 || strings.HasPrefix(fields[len(fields)-1], "*") {
		if len(fields) > 0 {
			fields = fields[:len(fields)-1]
		}
	}
	if len(fields) == 0 {
		return nil
	}
	command := strings.ToUpper(fields[0])
	if command == "EXCLUDE_OBJECT_DEFINE" || command == "EXCLUDE_OBJECT_START" || command == "EXCLUDE_OBJECT_END" {
		// Klipper exclude-object commands accompany, rather than replace, Orca's
		// ordinary object-label comments. They do not change XYZ/E coordinates.
		return nil
	}
	if command == "PRINT_START" || command == "PRINT_END" {
		p.warnOnce("Printer macros are not expanded (for example PRINT_START/PRINT_END); only explicit G-code motion is represented")
		return nil
	}
	if command == "SET_RETRACTION" || command == "SET_PRESSURE_ADVANCE" {
		// Like SET_VELOCITY_LIMIT, these exact configuration commands do not
		// encode motion. Leave their firmware-specific arguments uninterpreted;
		// accepting them here is not firmware syntax or value validation.
		p.warnOnce(command + " firmware effects are not simulated; only explicit G-code motion is represented")
		return nil
	}
	if command == "SET_VELOCITY_LIMIT" {
		p.warnOnce("Klipper velocity/acceleration commands are not expanded; coordinates and commanded feedrates are still read from explicit motion")
		return nil
	}
	if command == "G4" || command == "G04" {
		return nil // Dwell parameters (normally P/S) do not change geometry.
	}
	if command == "G28" {
		return p.home(fields[1:])
	}
	if nonMotionMCommands[command] {
		return nil
	}
	if strings.HasPrefix(command, "M") && command != "M82" && command != "M83" {
		return fmt.Errorf("unsupported M-code command %s", command)
	}
	allowedWords := "XYZEFIJRH"
	if command == "G2" || command == "G02" || command == "G3" || command == "G03" {
		allowedWords += "P"
	}
	args, err := parseMotionWords(fields[1:], allowedWords)
	if err != nil {
		return err
	}
	switch command {
	case "G0", "G00", "G1", "G01":
		return p.linearMove(args)
	case "G2", "G02", "G3", "G03":
		return p.arcMove(args, command == "G2" || command == "G02")
	case "G17":
		return nil // XY arc plane.
	case "G18", "G19":
		return fmt.Errorf("only the XY arc plane (G17) is supported")
	case "G20":
		p.unitsScale = 25.4
		return nil
	case "G21":
		p.unitsScale = 1
		return nil
	case "G90":
		p.globalRelative = false
		return nil
	case "G91":
		p.globalRelative = true
		return nil
	case "G92":
		return p.setPosition(args)
	case "M82":
		p.extruderRelative = false
		return nil
	case "M83":
		p.extruderRelative = true
		return nil
	}
	if strings.HasPrefix(command, "T") {
		tool, err := strconv.Atoi(command[1:])
		if err != nil || tool < 0 || tool > 254 {
			return fmt.Errorf("invalid or unsupported tool selector %q", command)
		}
		p.currentTool = tool
		return nil
	}
	if strings.HasPrefix(command, "G") {
		return fmt.Errorf("unsupported G-code command %s", command)
	}
	return fmt.Errorf("unsupported command %s", command)
}

func (p *toolpathParser) home(fields []string) error {
	homed := [3]bool{}
	for _, field := range fields {
		upper := strings.ToUpper(field)
		compactAxes := upper != ""
		for _, axis := range upper {
			if axis < 'X' || axis > 'Z' {
				compactAxes = false
				break
			}
		}
		if compactAxes {
			for _, axis := range upper {
				homed[axis-'X'] = true
			}
			continue
		}
		if len(upper) < 2 || upper[0] < 'X' || upper[0] > 'Z' {
			return fmt.Errorf("unsupported G28 word %q", field)
		}
		value, err := strconv.ParseFloat(upper[1:], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("invalid G28 word %q", field)
		}
		homed[upper[0]-'X'] = true
	}
	if len(fields) == 0 {
		homed = [3]bool{true, true, true}
	}
	for axis, found := range homed {
		if found {
			p.position[axis] = 0
			p.origin[axis] = 0
		}
	}
	p.warnOnce("G28 homing endpoints are assumed to be zero for visualization; physical homing is printer-specific")
	return nil
}

func (p *toolpathParser) warnOnce(warning string) {
	for _, existing := range p.preview.Warnings {
		if existing == warning {
			return
		}
	}
	p.preview.Warnings = append(p.preview.Warnings, warning)
}

func parseMotionWords(fields []string, allowedWords string) (map[byte]float64, error) {
	args := make(map[byte]float64, len(fields))
	for _, field := range fields {
		if len(field) < 2 {
			return nil, fmt.Errorf("invalid command word %q", field)
		}
		letter := field[0]
		if !strings.ContainsRune(allowedWords, rune(letter)) {
			return nil, fmt.Errorf("unsupported command word %q", field)
		}
		if _, duplicate := args[letter]; duplicate {
			return nil, fmt.Errorf("duplicate %c word", letter)
		}
		value, err := strconv.ParseFloat(field[1:], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("invalid numeric word %q", field)
		}
		args[letter] = value
	}
	return args, nil
}

// Recognize only the actual Orca/Prusa PNG thumbnail envelope, not the outer
// THUMBNAIL_BLOCK_START/END metadata. No image is decoded or buffered: the
// declared encoded length and base64 alphabet are checked within source limits.
func (p *toolpathParser) beginThumbnail(comment string) error {
	fields := strings.Fields(comment)
	if len(fields) != 4 || fields[0] != "thumbnail" || fields[1] != "begin" {
		return fmt.Errorf("malformed or unexpected thumbnail marker")
	}
	dimensions := strings.Split(fields[2], "x")
	if len(dimensions) != 2 {
		return fmt.Errorf("invalid thumbnail dimensions")
	}
	for _, dimension := range dimensions {
		if _, err := parseThumbnailPositiveInt(dimension); err != nil {
			return err
		}
	}
	size, err := parseThumbnailPositiveInt(fields[3])
	if err != nil || size%4 != 0 {
		return fmt.Errorf("invalid thumbnail encoded length")
	}
	p.thumbnailActive = true
	p.thumbnailLeft = size
	p.thumbnailPadding = 0
	return nil
}

func parseThumbnailPositiveInt(word string) (int, error) {
	for _, char := range word {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("invalid thumbnail integer")
		}
	}
	value, err := strconv.Atoi(word)
	if err != nil || value <= 0 || value > maxToolpathSourceBytes {
		return 0, fmt.Errorf("thumbnail integer outside preview limits")
	}
	return value, nil
}

func (p *toolpathParser) parseThumbnailPayload(comment string) error {
	if comment == "thumbnail end" {
		if p.thumbnailLeft != 0 {
			return fmt.Errorf("thumbnail payload length does not match declaration")
		}
		p.thumbnailActive = false
		return nil
	}
	if comment == "" || len(comment) > p.thumbnailLeft {
		return fmt.Errorf("empty or oversized thumbnail payload")
	}
	for _, char := range comment {
		if char == '=' {
			p.thumbnailPadding++
			if p.thumbnailPadding > 2 || p.thumbnailLeft > 2 {
				return fmt.Errorf("invalid thumbnail base64 padding")
			}
		} else if p.thumbnailPadding != 0 || !(char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '+' || char == '/') {
			return fmt.Errorf("invalid thumbnail base64 payload or nested marker")
		}
		p.thumbnailLeft--
	}
	return nil
}

func (p *toolpathParser) parseComment(comment string) error {
	if fields := strings.Fields(comment); len(fields) > 0 && fields[0] == "thumbnail" {
		return p.beginThumbnail(comment)
	}
	if strings.HasPrefix(comment, "V") {
		return fmt.Errorf("embedded comment commands (;V...) are not supported")
	}
	switch comment {
	case "LAYER_CHANGE", "CHANGE_LAYER":
		p.sawLayerMarker = true
		p.preview.LayerSemantics = "markers"
		p.beginLayer(p.position[2])
		p.pendingLayerZ = true
		return nil
	case "WIPE_START":
		p.wiping = true
		return nil
	case "WIPE_END":
		p.wiping = false
		return nil
	}
	for _, prefix := range []string{"TYPE:", "FEATURE:"} {
		if strings.HasPrefix(comment, prefix) {
			p.currentFeature = strings.TrimSpace(strings.TrimPrefix(comment, prefix))
			return nil
		}
	}
	for _, prefix := range []string{"Z:", "Z_HEIGHT:"} {
		if strings.HasPrefix(comment, prefix) {
			z, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(comment, prefix)), 64)
			if err != nil || math.IsNaN(z) || math.IsInf(z, 0) {
				return fmt.Errorf("invalid layer Z marker")
			}
			if p.currentLayer < 0 {
				p.beginLayer(z)
			} else if p.pendingLayerZ {
				if p.retainedLayer >= 0 {
					p.preview.Layers[p.retainedLayer].Z = z
				}
				p.pendingLayerZ = false
			} else if !p.sawLayerMarker {
				p.beginLayer(z)
			}
			return nil
		}
	}
	if strings.HasPrefix(comment, "printing object ") {
		label := strings.TrimPrefix(comment, "printing object ")
		matches := objectLabelPattern.FindStringSubmatch(label)
		if len(matches) != 4 || matches[1] == "" {
			return fmt.Errorf("malformed object label")
		}
		p.currentObject = matches[1]
		return nil
	}
	if strings.HasPrefix(comment, "stop printing object ") {
		p.currentObject = ""
		return nil
	}
	return nil
}

func (p *toolpathParser) setPosition(args map[byte]float64) error {
	if len(args) == 0 {
		p.origin = p.position
		p.extruderOrigin = p.extruderPosition
		return nil
	}
	for axis := 0; axis < 3; axis++ {
		if value, found := args[byte('X'+axis)]; found {
			p.origin[axis] = p.position[axis] - value*p.unitsScale
		}
	}
	if value, found := args['E']; found {
		// G92 changes the logical coordinate origin, not the physical extruder
		// position. Subsequent absolute E words must be measured from that origin.
		p.extruderOrigin = p.extruderPosition - value*p.unitsScale
	}
	for letter := range args {
		if !strings.ContainsRune("XYZE", rune(letter)) {
			return fmt.Errorf("unsupported G92 axis %c", letter)
		}
	}
	return nil
}

func (p *toolpathParser) target(args map[byte]float64) ([3]float64, float64, error) {
	target := p.position
	for axis := 0; axis < 3; axis++ {
		if value, found := args[byte('X'+axis)]; found {
			value *= p.unitsScale
			if p.globalRelative {
				target[axis] += value
			} else {
				target[axis] = p.origin[axis] + value
			}
		}
	}
	extruder := p.extruderPosition
	if value, found := args['E']; found {
		value *= p.unitsScale
		if p.globalRelative || p.extruderRelative {
			extruder += value
		} else {
			extruder = p.extruderOrigin + value
		}
	}
	for _, value := range []float64{target[0], target[1], target[2], extruder} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return target, extruder, fmt.Errorf("non-finite target position")
		}
	}
	return target, extruder, nil
}

func (p *toolpathParser) updateFeedrate(args map[byte]float64) {
	if value, found := args['F']; found {
		p.feedrate = value * p.unitsScale / 60
	}
}

func (p *toolpathParser) linearMove(args map[byte]float64) error {
	target, extruder, err := p.target(args)
	if err != nil {
		return err
	}
	p.updateFeedrate(args)
	deltaE := extruder - p.extruderPosition
	if deltaE > 0 && (target[0] != p.position[0] || target[1] != p.position[1]) {
		p.recordSegment(p.position, target, deltaE)
	}
	p.position = target
	p.extruderPosition = extruder
	return nil
}

func (p *toolpathParser) arcMove(args map[byte]float64, clockwise bool) error {
	// Orca 2.5.0 GCodeWriter::_spiral_travel_to_z emits P1 for a single
	// spiral lift. Klipper's klippy/extras/gcode_arcs.py does not read P:
	// the XY endpoint determines the sweep (one turn for coincident XY),
	// while Z is interpolated. Accept only this compatibility value; other
	// values have firmware-dependent multi-turn semantics, not simulated here.
	if turns, found := args['P']; found && turns != 1 {
		return fmt.Errorf("only P1 is supported on G2/G3 arcs")
	}
	target, extruder, err := p.target(args)
	if err != nil {
		return err
	}
	p.updateFeedrate(args)
	start := p.position
	_, hasR := args['R']
	_, hasI := args['I']
	_, hasJ := args['J']
	if hasR && (hasI || hasJ) {
		return fmt.Errorf("arc must use either R or I/J, not both")
	}
	if !hasR && !hasI && !hasJ {
		return fmt.Errorf("arc requires R or I/J")
	}

	var centerX, centerY float64
	if hasI || hasJ {
		centerX = start[0] + args['I']*p.unitsScale
		centerY = start[1] + args['J']*p.unitsScale
	} else {
		radius := args['R'] * p.unitsScale
		if radius == 0 || (start[0] == target[0] && start[1] == target[1]) {
			return fmt.Errorf("degenerate R arc")
		}
		dx, dy := target[0]-start[0], target[1]-start[1]
		chord := math.Hypot(dx, dy)
		absoluteRadius := math.Abs(radius)
		if chord > 2*absoluteRadius+1e-7 {
			return fmt.Errorf("arc radius is too small for its chord")
		}
		height := math.Sqrt(math.Max(0, absoluteRadius*absoluteRadius-chord*chord/4))
		midX, midY := (start[0]+target[0])/2, (start[1]+target[1])/2
		perpX, perpY := -dy/chord, dx/chord
		candidates := [][2]float64{{midX + height*perpX, midY + height*perpY}, {midX - height*perpX, midY - height*perpY}}
		found := false
		for _, candidate := range candidates {
			sweep := arcSweep(start, target, candidate[0], candidate[1], clockwise, false)
			if (radius > 0 && math.Abs(sweep) <= math.Pi+1e-7) || (radius < 0 && math.Abs(sweep) >= math.Pi-1e-7) {
				centerX, centerY = candidate[0], candidate[1]
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("could not resolve arc center")
		}
	}

	radius := math.Hypot(start[0]-centerX, start[1]-centerY)
	if radius == 0 || math.IsNaN(radius) || math.IsInf(radius, 0) {
		return fmt.Errorf("zero-radius or invalid arc")
	}
	endRadius := math.Hypot(target[0]-centerX, target[1]-centerY)
	if math.Abs(endRadius-radius) > math.Max(0.05, radius*0.001) {
		return fmt.Errorf("arc endpoints have inconsistent radii")
	}
	fullCircle := start[0] == target[0] && start[1] == target[1]
	if fullCircle && hasR {
		return fmt.Errorf("full-circle R arc is ambiguous")
	}
	sweep := arcSweep(start, target, centerX, centerY, clockwise, fullCircle)
	maxAngle := math.Pi / 2
	if radius > toolpathArcToleranceMM {
		maxAngle = 2 * math.Acos(1-toolpathArcToleranceMM/radius)
	}
	pieces := int(math.Ceil(math.Abs(sweep) / maxAngle))
	if pieces < 1 {
		pieces = 1
	}
	if pieces > maxToolpathArcSegments {
		return fmt.Errorf("arc requires %d segments, exceeding the %d-segment limit", pieces, maxToolpathArcSegments)
	}

	startAngle := math.Atan2(start[1]-centerY, start[0]-centerX)
	deltaE := extruder - p.extruderPosition
	if deltaE <= 0 || (!fullCircle && start[0] == target[0] && start[1] == target[1]) {
		p.position = target
		p.extruderPosition = extruder
		return nil
	}
	p.arcWork += pieces
	if p.arcWork > maxToolpathArcWork {
		return fmt.Errorf("arcs require more than the %d-segment total preview work limit", maxToolpathArcWork)
	}
	previous := start
	for index := 1; index <= pieces; index++ {
		if index%256 == 0 {
			select {
			case <-p.ctx.Done():
				return p.ctx.Err()
			default:
			}
		}
		fraction := float64(index) / float64(pieces)
		next := [3]float64{target[0], target[1], start[2] + (target[2]-start[2])*fraction}
		if index != pieces {
			angle := startAngle + sweep*fraction
			next[0] = centerX + radius*math.Cos(angle)
			next[1] = centerY + radius*math.Sin(angle)
		}
		if deltaE > 0 && (next[0] != previous[0] || next[1] != previous[1]) {
			p.recordSegment(previous, next, deltaE/float64(pieces))
		}
		previous = next
	}
	p.position = target
	p.extruderPosition = extruder
	return nil
}

func arcSweep(start, end [3]float64, centerX, centerY float64, clockwise, fullCircle bool) float64 {
	if fullCircle {
		if clockwise {
			return -2 * math.Pi
		}
		return 2 * math.Pi
	}
	startAngle := math.Atan2(start[1]-centerY, start[0]-centerX)
	endAngle := math.Atan2(end[1]-centerY, end[0]-centerX)
	sweep := endAngle - startAngle
	if clockwise {
		if sweep >= 0 {
			sweep -= 2 * math.Pi
		}
	} else if sweep <= 0 {
		sweep += 2 * math.Pi
	}
	return sweep
}

func (p *toolpathParser) beginLayer(z float64) {
	p.currentLayer++
	p.preview.TotalLayers++
	p.retainedLayer = -1
	if len(p.preview.Layers) < maxToolpathLayers {
		p.preview.Layers = append(p.preview.Layers, ToolpathLayer{Index: p.currentLayer, Z: z, Segments: make([]ToolpathSegment, 0)})
		p.retainedLayer = len(p.preview.Layers) - 1
	}
}

func (p *toolpathParser) recordSegment(from, to [3]float64, extrusion float64) {
	if p.currentLayer < 0 || (!p.sawLayerMarker && (math.IsNaN(p.fallbackZ) || math.Abs(from[2]-p.fallbackZ) > 1e-9)) {
		p.beginLayer(from[2])
		p.fallbackZ = from[2]
	}
	p.preview.TotalSegments++
	switch p.currentFeature {
	case "Support", "Support interface", "Support transition":
		p.preview.SupportSegments++
	case "Brim":
		p.preview.BrimSegments++
	}
	// Bounds describe all positive-extrusion XY geometry in the completely read
	// source, including geometry omitted from the retained prefix.
	p.includeInBounds(from)
	p.includeInBounds(to)
	if p.retainedLayer < 0 || p.retainedSegments >= maxToolpathSegments {
		return
	}
	feature := p.currentFeature
	if p.wiping {
		feature = "Wipe"
	}
	segment := ToolpathSegment{From: from, To: to, Extrusion: extrusion, Feedrate: p.feedrate, Feature: feature, Object: p.currentObject, Tool: p.currentTool}
	layer := &p.preview.Layers[p.retainedLayer]
	layer.Segments = append(layer.Segments, segment)
	p.retainedSegments++
}

func (p *toolpathParser) includeInBounds(point [3]float64) {
	if p.preview.Bounds == nil {
		p.preview.Bounds = &ToolpathBounds{Min: point, Max: point}
		return
	}
	for axis := 0; axis < 3; axis++ {
		p.preview.Bounds.Min[axis] = math.Min(p.preview.Bounds.Min[axis], point[axis])
		p.preview.Bounds.Max[axis] = math.Max(p.preview.Bounds.Max[axis], point[axis])
	}
}
