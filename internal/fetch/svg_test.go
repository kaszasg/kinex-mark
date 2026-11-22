package fetch

import "testing"

const svgOpen = `<svg xmlns="http://www.w3.org/2000/svg">`

// appIconSVG is the application icon of the design rules (GRAPHICS.md 7.7),
// a realistic icon with role, aria-label, title and a transformed group.
const appIconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" role="img" aria-label="Kinexium"><title>Kinexium</title><path d="M0 0H106L128 22V128H22L0 106Z" fill="#10212a"/><g transform="translate(26 26) scale(1.19)"><path d="M16 8V56M16 38 45 13M27.5 28 45 51" stroke="#d4dcd5" stroke-width="6.4" fill="none"/><circle cx="47" cy="11.5" r="6.8" fill="#77dbed"/><circle cx="47" cy="53" r="6.8" fill="#d4dcd5"/></g></svg>`

func TestIsSafeSVGAccepts(t *testing.T) {
	t.Parallel()
	for name, s := range map[string]string{
		"minimal":         svgOpen + `</svg>`,
		"app icon":        appIconSVG,
		"xml declaration": "<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"no\"?>\n<!-- Generator: test -->\n" + svgOpen + "<rect width=\"1\" height=\"1\"/></svg>\n<!-- trailer -->",
		"gradient and internal refs": `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 16 16">` +
			`<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#fff"/><stop offset="1" stop-color="#000"/></linearGradient>` +
			`<radialGradient id="h" xlink:href="#g"/><symbol id="s"><circle r="4"/></symbol><clipPath id="c"><rect width="8" height="8"/></clipPath></defs>` +
			`<rect width="16" height="16" fill="url(#g)" clip-path="url( '#c' )"/><use href="#s" x="8" y="8"/><use xlink:href="#s"/></svg>`,
		"style element":   svgOpen + `<style>.a{fill:url(#g);stroke:#000}/* note */@media (prefers-color-scheme:dark){.a{fill:#fff}}</style><path class="a" d="M0 0h1v1z"/></svg>`,
		"style cdata":     svgOpen + `<style><![CDATA[.a>.b{fill:red}]]></style></svg>`,
		"style attribute": svgOpen + `<rect style="fill: url(&quot;#g&quot;); opacity: .5" width="1" height="1"/></svg>`,
		"filter":          svgOpen + `<filter id="f"><feGaussianBlur in="SourceGraphic" stdDeviation="2" result="b"/><feMerge><feMergeNode in="b"/></feMerge></filter><g filter="url(#f)"/></svg>`,
		"text":            svgOpen + `<text x="0" y="10" font-family="sans-serif" xml:space="preserve">K &amp; M &lt;3</text></svg>`,
		"byte order mark": "\xef\xbb\xbf" + svgOpen + `</svg>`,
	} {
		if !isSafeSVG([]byte(s)) {
			t.Errorf("%s: rejected", name)
		}
	}
}

// TestIsSafeSVGRejectsBypasses covers content the earlier substring check
// let through, plus the structural rules of svgStructureOK.
func TestIsSafeSVGRejectsBypasses(t *testing.T) {
	t.Parallel()
	for name, s := range map[string]string{
		// encoded javascript: (character references are decoded by the parser)
		"decimal reference in href":   svgOpen + `<use href="&#106;avascript:alert(1)"/></svg>`,
		"hex reference in xlink href": `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><use xlink:href="&#x6A;avascript:alert(1)"/></svg>`,
		"tab inside scheme":           svgOpen + `<use href="#a" fill="url(java&#9;script:alert(1))"/></svg>`,
		"encoded link element":        svgOpen + `<a href="&#106;avascript:alert(1)"><rect width="9" height="9"/></a></svg>`,
		"set to encoded javascript":   svgOpen + `<set attributeName="href" to="&#106;avascript:alert(1)"/></svg>`,
		"animate values":              svgOpen + `<animate attributeName="href" values="&#106;avascript:alert(1)"/></svg>`,
		"encoded style url":           svgOpen + `<rect style="fill:url(&#x6A;avascript:x)"/></svg>`,
		// foreign content and scripts in other namespaces
		"foreignObject with iframe":   svgOpen + `<foreignObject width="9" height="9"><iframe xmlns="http://www.w3.org/1999/xhtml" src="&#106;avascript:alert(1)"/></foreignObject></svg>`,
		"prefixed xhtml script":       `<svg xmlns="http://www.w3.org/2000/svg" xmlns:h="http://www.w3.org/1999/xhtml"><h:script>alert(1)</h:script></svg>`,
		"default namespace switch":    svgOpen + `<g xmlns="http://www.w3.org/1999/xhtml"><p>x</p></g></svg>`,
		"unknown namespace element":   `<svg xmlns="http://www.w3.org/2000/svg" xmlns:s="http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd"><s:namedview/></svg>`,
		"unknown namespace attribute": `<svg xmlns="http://www.w3.org/2000/svg" xmlns:i="http://www.inkscape.org/namespaces/inkscape"><g i:label="x"/></svg>`,
		"no namespace":                `<svg><rect/></svg>`,
		"event handler via reference": svgOpen + `<rect width="1" height="1" o&#110;click="alert(1)"/></svg>`,
		"unknown attribute":           svgOpen + `<rect formaction="x"/></svg>`,
		// entity and DOCTYPE declarations
		"internal entity":   `<?xml version="1.0"?><!DOCTYPE svg [<!ENTITY x "&#60;script&#62;alert(1)&#60;/script&#62;">]>` + svgOpen + `&x;</svg>`,
		"external doctype":  `<?xml version="1.0"?><!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">` + svgOpen + `</svg>`,
		"undeclared entity": svgOpen + `<text>&nbsp;</text></svg>`,
		// external references
		"image element":          svgOpen + `<image href="https://evil.example/x.png"/></svg>`,
		"feImage":                svgOpen + `<filter id="f"><feImage href="https://evil.example/x.png"/></filter></svg>`,
		"external use":           svgOpen + `<use href="https://evil.example/s.svg#a"/></svg>`,
		"external fill":          svgOpen + `<rect fill="url(https://evil.example/p.svg#p)"/></svg>`,
		"data url fill":          svgOpen + `<rect fill="url(data:image/svg+xml;base64,AAAA)"/></svg>`,
		"style import":           svgOpen + `<style>@import url(https://evil.example/x.css);</style></svg>`,
		"style external url":     svgOpen + `<style>rect{fill:url("https://evil.example/p.svg#p")}</style></svg>`,
		"style escape":           svgOpen + `<style>rect{fill:u\72l(https://evil.example/p)}</style></svg>`,
		"style comment split":    svgOpen + `<style>rect{fill:u/**/rl(https://evil.example/p)}</style></svg>`,
		"style cdata split":      svgOpen + `<style>rect{fill:u<![CDATA[rl(https://evil.example/p)]]>}</style></svg>`,
		"style image-set":        svgOpen + `<style>rect{fill:image-set("https://evil.example/p" 1x)}</style></svg>`,
		"markup inside style":    svgOpen + `<style><rect/></style></svg>`,
		"stylesheet instruction": `<?xml version="1.0"?><?xml-stylesheet href="https://evil.example/x.css"?>` + svgOpen + `</svg>`,
		// malformed or multi-root documents
		"unclosed element":   svgOpen + `<g></svg>`,
		"truncated":          svgOpen + `<rect`,
		"second root":        svgOpen + `</svg><svg xmlns="http://www.w3.org/2000/svg"></svg>`,
		"text after root":    svgOpen + `</svg>trailing`,
		"other root":         `<svg-like xmlns="http://www.w3.org/2000/svg"/>`,
		"non utf-8 encoding": `<?xml version="1.0" encoding="ISO-8859-1"?>` + svgOpen + `</svg>`,
	} {
		if isSafeSVG([]byte(s)) {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestCSSSafe(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]bool{
		"#fff":                       true,
		"url(#a)":                    true,
		"url( \"#a\" ) url('#b')":    true,
		"M0 0h1v1z":                  true,
		"url(#a) url(x.svg)":         false,
		"URL(HTTPS://x)":             false,
		"JaVaScRiPt:alert(1)":        false,
		"java\nscript:alert(1)":      false,
		"/* open comment url(http:x": true,
		"@IMPORT 'x.css'":            false,
		"\\75 rl(x)":                 false,
	} {
		if got := cssSafe(in); got != want {
			t.Errorf("cssSafe(%q) = %v, want %v", in, got, want)
		}
	}
}
