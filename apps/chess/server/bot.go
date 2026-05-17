package server

import "strings"

// isLinkPreviewBot reports whether the request's User-Agent looks like a
// link-preview / crawler bot (WhatsApp, Slackbot, Twitter, Facebook, Discord,
// Telegram, Skype, LinkedIn, generic bots). These clients fetch invite URLs
// to generate previews — they must never trigger side effects like claiming
// a participant token.
func isLinkPreviewBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	if ua == "" {
		return true
	}
	needles := []string{
		"whatsapp",
		"slackbot",
		"slack-imgproxy",
		"twitterbot",
		"facebookexternalhit",
		"facebookcatalog",
		"discordbot",
		"telegrambot",
		"skypeuripreview",
		"linkedinbot",
		"redditbot",
		"applebot",
		"bingbot",
		"googlebot",
		"yandexbot",
		"duckduckbot",
		"baiduspider",
		"embedly",
		"pinterest",
		"vkshare",
		"w3c_validator",
		"http.rb",
		"go-http-client",
		"python-requests",
		"curl/",
		"wget/",
		"node-fetch",
		"axios",
		"headlesschrome",
		"prerender",
		"snapchat",
	}
	for _, n := range needles {
		if strings.Contains(ua, n) {
			return true
		}
	}
	if strings.Contains(ua, "bot") || strings.Contains(ua, "spider") || strings.Contains(ua, "crawler") {
		return true
	}
	return false
}
