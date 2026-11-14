package domain

import "wwhsrt/pkg/gameapi"

func constructionEstimate(a gameapi.Alignment) (gameapi.ConstructionEstimate, error) {
	intervals := a.CostIntervals
	if len(intervals) == 0 {
		// Frozen legacy Japanese works preserve their original quotes and rounding.
		intervals = []gameapi.ConstructionInterval{{Country: "Japan", ProfileID: "japan", BasePerKM: 2_000_000_000, SurfaceKM: a.SurfaceKM, HillKM: a.HillKM, ViaductKM: a.ViaductKM, TunnelKM: a.TunnelKM, HillFactorPercent: 150, TunnelFactor: 5}}
	}
	var e gameapi.ConstructionEstimate
	surface, hills, bridges, tunnels := 0, 0, 0, 0
	if len(intervals) > 16 {
		return e, errCode("invalid_scenario")
	}
	for _, p := range intervals {
		if p.Country == "" || p.ProfileID == "" || p.BasePerKM < 300_000_000 || p.BasePerKM > 10_000_000_000 || p.SurfaceKM < 0 || p.HillKM < 0 || p.ViaductKM < 0 || p.TunnelKM < 0 || p.SurfaceKM+p.HillKM+p.ViaductKM+p.TunnelKM > 6000 || p.HillFactorPercent < 100 || p.HillFactorPercent > 400 || p.TunnelFactor < 5 || p.TunnelFactor > 12 {
			return e, errCode("invalid_scenario")
		}
		surface += p.SurfaceKM
		hills += p.HillKM
		bridges += p.ViaductKM
		tunnels += p.TunnelKM
		systems := gameapi.Money(300_000_000)
		civil := p.BasePerKM - systems
		e.Surface += gameapi.Money(p.SurfaceKM) * p.BasePerKM
		e.Hills += gameapi.Money(p.HillKM) * (systems + civil*gameapi.Money(p.HillFactorPercent)/100)
		e.Viaducts += gameapi.Money(p.ViaductKM) * (systems + civil*250/100)
		e.Tunnels += gameapi.Money(p.TunnelKM) * (systems + civil*gameapi.Money(p.TunnelFactor))
	}
	if surface != a.SurfaceKM || hills != a.HillKM || bridges != a.ViaductKM || tunnels != a.TunnelKM {
		return e, errCode("invalid_scenario")
	}
	return e, nil
}
