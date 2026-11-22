package fetch

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

// Namespaces the SVG structure check knows.
const (
	svgNS   = "http://www.w3.org/2000/svg"
	xlinkNS = "http://www.w3.org/1999/xlink"
	xmlNS   = "http://www.w3.org/XML/1998/namespace"
)

// svgElements is the allow-list of SVG elements an icon may contain. Not
// listed, so rejected: script, foreignObject, a, image, feImage, the
// animation elements (set, animate, animateMotion, animateTransform, mpath),
// font elements and anything outside the SVG namespace.
var svgElements = setOf(
	"svg", "g", "defs", "symbol", "use", "title", "desc", "style",
	"path", "rect", "circle", "ellipse", "line", "polyline", "polygon",
	"text", "tspan", "textPath",
	"linearGradient", "radialGradient", "stop", "pattern", "clipPath", "mask", "marker",
	"filter", "feBlend", "feColorMatrix", "feComponentTransfer", "feComposite",
	"feConvolveMatrix", "feDiffuseLighting", "feDisplacementMap", "feDistantLight",
	"feDropShadow", "feFlood", "feFuncA", "feFuncB", "feFuncG", "feFuncR",
	"feGaussianBlur", "feMerge", "feMergeNode", "feMorphology", "feOffset",
	"fePointLight", "feSpecularLighting", "feSpotLight", "feTile", "feTurbulence",
)

// svgAttributes is the allow-list of attributes without a namespace prefix:
// geometry, presentation, gradient, filter and accessibility attributes.
// Event handlers (on...) are not listed. href is listed but must be a
// same-document reference (#id).
var svgAttributes = setOf(
	// core and accessibility
	"id", "class", "style", "lang", "tabindex", "role", "focusable",
	"aria-label", "aria-labelledby", "aria-describedby", "aria-hidden",
	"version", "baseProfile", "viewBox", "preserveAspectRatio", "href",
	"type", "media",
	// geometry
	"x", "y", "x1", "y1", "x2", "y2", "cx", "cy", "r", "rx", "ry", "fx", "fy", "fr",
	"width", "height", "d", "points", "transform", "pathLength",
	"dx", "dy", "rotate", "textLength", "lengthAdjust", "startOffset", "method", "spacing", "side",
	// presentation
	"fill", "fill-opacity", "fill-rule", "stroke", "stroke-width", "stroke-linecap",
	"stroke-linejoin", "stroke-miterlimit", "stroke-dasharray", "stroke-dashoffset",
	"stroke-opacity", "opacity", "clip-path", "clip-rule", "mask", "filter", "color",
	"display", "visibility", "overflow", "stop-color", "stop-opacity",
	"flood-color", "flood-opacity", "lighting-color", "color-interpolation",
	"color-interpolation-filters", "color-rendering", "shape-rendering", "text-rendering",
	"image-rendering", "vector-effect", "paint-order", "mix-blend-mode", "isolation",
	"enable-background", "marker-start", "marker-mid", "marker-end",
	"font-family", "font-size", "font-size-adjust", "font-style", "font-variant",
	"font-weight", "font-stretch", "text-anchor", "text-decoration", "dominant-baseline",
	"alignment-baseline", "baseline-shift", "letter-spacing", "word-spacing",
	"writing-mode", "direction", "unicode-bidi", "white-space",
	// gradients, patterns, clipping, masking, markers
	"offset", "gradientUnits", "gradientTransform", "spreadMethod",
	"patternUnits", "patternContentUnits", "patternTransform",
	"clipPathUnits", "maskUnits", "maskContentUnits",
	"markerUnits", "markerWidth", "markerHeight", "refX", "refY", "orient",
	// filters
	"filterUnits", "primitiveUnits", "in", "in2", "result", "stdDeviation", "mode",
	"operator", "k1", "k2", "k3", "k4", "values", "tableValues", "slope", "intercept",
	"amplitude", "exponent", "scale", "xChannelSelector", "yChannelSelector", "radius",
	"baseFrequency", "numOctaves", "seed", "stitchTiles", "order", "kernelMatrix",
	"divisor", "bias", "targetX", "targetY", "edgeMode", "preserveAlpha",
	"kernelUnitLength", "surfaceScale", "diffuseConstant", "specularConstant",
	"specularExponent", "azimuth", "elevation", "z", "pointsAtX", "pointsAtY",
	"pointsAtZ", "limitingConeAngle", "edgeModes",
)

func setOf(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// svgStructureOK parses data with the strict XML tokenizer and accepts it
// only when every token is allowed: an optional leading <?xml ...?>
// declaration, comments, a single <svg> root in the SVG namespace, elements
// and attributes from the allow-lists, and no DOCTYPE or entity declaration,
// other processing instruction, undeclared entity or external reference.
// Character references are decoded by the tokenizer before the checks, so
// encoded forms of javascript: or url(...) are caught.
func svgStructureOK(data []byte) bool {
	d := xml.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
	d.Strict = true
	// d.Entity stays nil: only the five predefined entities are accepted.
	// d.CharsetReader stays nil: an encoding other than UTF-8 is rejected.
	var (
		depth     int
		rootSeen  bool
		first     = true
		inStyle   bool
		styleText strings.Builder
	)
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return rootSeen && depth == 0
		}
		if err != nil {
			return false
		}
		isFirst := first
		first = false
		switch t := tok.(type) {
		case xml.ProcInst:
			if !isFirst || t.Target != "xml" {
				return false
			}
		case xml.Directive:
			return false // DOCTYPE, ENTITY and the like
		case xml.Comment:
		case xml.CharData:
			switch {
			case inStyle:
				styleText.Write(t)
			case depth == 0 && len(bytes.TrimSpace(t)) > 0:
				return false
			}
		case xml.StartElement:
			if inStyle || (depth == 0 && rootSeen) {
				return false // markup inside <style>, or a second root
			}
			if t.Name.Space != svgNS || !svgElements[t.Name.Local] {
				return false
			}
			if depth == 0 && t.Name.Local != "svg" {
				return false
			}
			for _, a := range t.Attr {
				if !svgAttrOK(a) {
					return false
				}
			}
			rootSeen = true
			depth++
			if t.Name.Local == "style" {
				inStyle = true
				styleText.Reset()
			}
		case xml.EndElement:
			if inStyle {
				if !cssSafe(styleText.String()) {
					return false
				}
				inStyle = false
			}
			depth--
		default:
			return false
		}
	}
}

// svgAttrOK checks one attribute name against the allow-list and its value
// against the reference rules.
func svgAttrOK(a xml.Attr) bool {
	switch a.Name.Space {
	case "xmlns":
		return true // prefix declaration; elements are checked by namespace
	case "":
		if a.Name.Local == "xmlns" {
			return true
		}
		if !svgAttributes[a.Name.Local] {
			return false
		}
		if a.Name.Local == "href" {
			return internalRef(a.Value)
		}
		return cssSafe(a.Value)
	case xlinkNS:
		return a.Name.Local == "href" && internalRef(a.Value)
	case xmlNS:
		return a.Name.Local == "space" || a.Name.Local == "lang"
	}
	return false
}

// internalRef accepts only same-document references such as "#grad".
func internalRef(v string) bool {
	v = strings.TrimSpace(v)
	return strings.HasPrefix(v, "#") && cssSafe(v)
}

// cssSafe rejects text that could reach outside the document or run code:
// CSS escapes, @import, url(...) other than url(#id), image-set() and
// friends, javascript: and data: in any letter case and with whitespace or
// control characters removed. CSS comments are removed first so they cannot
// split a keyword.
func cssSafe(s string) bool {
	s = stripCSSComments(s)
	s = strings.Map(func(r rune) rune {
		if r <= ' ' || r == 0x7f {
			return -1
		}
		return r
	}, strings.ToLower(s))
	if strings.ContainsRune(s, '\\') {
		return false
	}
	for _, bad := range []string{
		"@import", "javascript:", "vbscript:", "data:", "expression(", "-moz-binding",
		"behavior:", "image(", "image-set(", "cross-fade(", "element(", "src(",
	} {
		if strings.Contains(s, bad) {
			return false
		}
	}
	for rest := s; ; {
		i := strings.Index(rest, "url(")
		if i < 0 {
			return true
		}
		rest = strings.TrimLeft(rest[i+len("url("):], `"'`)
		if !strings.HasPrefix(rest, "#") {
			return false
		}
	}
}

// stripCSSComments removes /* ... */ comments; an unterminated comment runs
// to the end, as in CSS.
func stripCSSComments(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "/*")
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		j := strings.Index(s[i+2:], "*/")
		if j < 0 {
			return b.String()
		}
		s = s[i+2+j+2:]
	}
}
