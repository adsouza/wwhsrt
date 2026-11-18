package ui

import "image/color"

func AppearanceMode(value string) string {
	if value == "light" || value == "dark" {
		return value
	}
	return "system"
}
func NextAppearance(value string) string {
	switch AppearanceMode(value) {
	case "system":
		return "light"
	case "light":
		return "dark"
	default:
		return "system"
	}
}
func IsDark(s State) bool {
	return AppearanceMode(s.Appearance) == "dark" || AppearanceMode(s.Appearance) == "system" && s.Dark
}

// Palette is presentation only and is shared with the semantic browser host.
type Palette struct {
	Background   color.NRGBA `json:"background"`
	Surface      color.NRGBA `json:"surface"`
	Control      color.NRGBA `json:"control"`
	Text         color.NRGBA `json:"text"`
	Muted        color.NRGBA `json:"muted"`
	Border       color.NRGBA `json:"border"`
	Accent       color.NRGBA `json:"accent"`
	Selected     color.NRGBA `json:"selected"`
	Hover        color.NRGBA `json:"hover"`
	Pressed      color.NRGBA `json:"pressed"`
	Disabled     color.NRGBA `json:"disabled"`
	Action       color.NRGBA `json:"action"`
	ActionHover  color.NRGBA `json:"action-hover"`
	OnAction     color.NRGBA `json:"on-action"`
	Warning      color.NRGBA `json:"warning"`
	WarningText  color.NRGBA `json:"warning-text"`
	WarningHover color.NRGBA `json:"warning-hover"`
	Water        color.NRGBA `json:"water"`
	Land         color.NRGBA `json:"land"`
	Coast        color.NRGBA `json:"coast"`
	Operating    color.NRGBA `json:"operating"`
	Existing     color.NRGBA `json:"existing"`
	Construction color.NRGBA `json:"construction"`
	Draft        color.NRGBA `json:"draft"`
	Outline      color.NRGBA `json:"outline"`
	TerrainFill  color.NRGBA `json:"terrain-fill"`
	TerrainText  color.NRGBA `json:"terrain-text"`
	Contour      color.NRGBA `json:"contour"`
}

func rgb(hex uint32) color.NRGBA {
	return color.NRGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 255}
}

func PaletteFor(s State) Palette {
	if IsDark(s) {
		return Palette{
			Background:   rgb(0x0f1d2e),
			Surface:      rgb(0x17283d),
			Control:      rgb(0x20364e),
			Text:         rgb(0xe7f0fa),
			Muted:        rgb(0xabc2da),
			Border:       rgb(0x6586a6),
			Accent:       rgb(0x8ec5ff),
			Selected:     rgb(0x1c3c59),
			Hover:        rgb(0x29435f),
			Pressed:      rgb(0x355474),
			Disabled:     rgb(0x33475e),
			Action:       rgb(0x1264d5),
			ActionHover:  rgb(0x0d50ad),
			OnAction:     rgb(0xffffff),
			Warning:      rgb(0x3f321d),
			WarningText:  rgb(0xffdc8b),
			WarningHover: rgb(0x554128),
			Water:        rgb(0x0c2438),
			Land:         rgb(0x263e35),
			Coast:        rgb(0x658a80),
			Existing:     rgb(0xa5b1c4),
			Operating:    rgb(0x4edab0),
			Construction: rgb(0xffc257),
			Draft:        rgb(0xff849d),
			Outline:      rgb(0x07121f),
			TerrainFill:  rgb(0x1b2e40),
			TerrainText:  rgb(0xe4d0a8),
			Contour:      rgb(0xc1b18a),
		}
	}
	return Palette{
		Background:   rgb(0xf2f9ff),
		Surface:      rgb(0xffffff),
		Control:      rgb(0xffffff),
		Text:         rgb(0x143252),
		Muted:        rgb(0x486581),
		Border:       rgb(0x6b8ba6),
		Accent:       rgb(0x1264d5),
		Selected:     rgb(0xe4f0ff),
		Hover:        rgb(0xe4f0ff),
		Pressed:      rgb(0xffe39a),
		Disabled:     rgb(0xdce1e6),
		Action:       rgb(0x1264d5),
		ActionHover:  rgb(0x143252),
		OnAction:     rgb(0xffffff),
		Warning:      rgb(0xfff3cf),
		WarningText:  rgb(0x704800),
		WarningHover: rgb(0xffe39a),
		Water:        rgb(0xcdeeff),
		Land:         rgb(0xe7f4cc),
		Coast:        rgb(0x6b8ba6),
		Existing:     rgb(0x59687b),
		Operating:    rgb(0x006e53),
		Construction: rgb(0xb97400),
		Draft:        rgb(0xb52d49),
		Outline:      rgb(0x143252),
		TerrainFill:  rgb(0xfffffa),
		TerrainText:  rgb(0x5a472a),
		Contour:      rgb(0x745632),
	}
}
