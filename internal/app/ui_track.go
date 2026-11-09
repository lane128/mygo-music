package app

import "github.com/egoist/mygo/ui"

// forTrack keeps unavailable list text distinct in either appearance without
// disabling a row's links, context menu or heart along with its play action.
func (p palette) forTrack(unavailable bool) palette {
	if !unavailable {
		return p
	}
	if p.dark {
		p.text, p.muted = ui.Hex("#68686f"), ui.Hex("#595960")
	} else {
		p.text, p.muted = ui.Hex("#a1a1a9"), ui.Hex("#afb0b8")
	}
	p.faint, p.hover = p.muted, ui.Transparent
	return p
}
