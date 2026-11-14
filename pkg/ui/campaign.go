package ui

import (
	"strconv"
	"strings"
	"wwhsrt/pkg/gameapi"
)

// CampaignFinished is persistent presentation state, independent of dismissed
// onboarding or the last command's transient status message.
func CampaignFinished(f gameapi.Frame) bool {
	return f.CampaignMonths > 0 && f.MonthsResolved >= f.CampaignMonths
}

func CampaignEndDetail(locale string, f gameapi.Frame) string {
	return strings.ReplaceAll(Text(locale, "campaignEndDetail"), "{years}", strconv.Itoa(f.CampaignMonths/12))
}
