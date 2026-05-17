package server

import "testing"

func TestIsLinkPreviewBot(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want bool
	}{
		// Real browsers — must NOT match.
		{"chrome desktop", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", false},
		{"safari ios", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1", false},
		{"firefox", "Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0", false},
		{"edge", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0", false},

		// Link-preview crawlers — MUST match.
		{"whatsapp", "WhatsApp/2.23.24.84 A", true},
		{"whatsapp capitalised", "WHATSAPP/2.0", true},
		{"slackbot link expander", "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)", true},
		{"slack image proxy", "Slack-ImgProxy 0.59 (+https://api.slack.com/robots)", true},
		{"twitter", "Twitterbot/1.0", true},
		{"facebook", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)", true},
		{"facebook catalog", "facebookcatalog/1.0", true},
		{"discord", "Mozilla/5.0 (compatible; Discordbot/2.0; +https://discordapp.com)", true},
		{"telegram", "TelegramBot (like TwitterBot)", true},
		{"skype", "SkypeUriPreview Preview/0.5", true},
		{"linkedin", "LinkedInBot/1.0 (compatible; Mozilla/5.0; Jakarta Commons-HttpClient/3.1 +http://www.linkedin.com)", true},
		{"google", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", true},
		{"bing", "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)", true},
		{"apple", "Mozilla/5.0 (compatible; Applebot/0.1; +http://www.apple.com/go/applebot)", true},

		// Generic markers.
		{"empty UA", "", true},
		{"generic bot", "SomeUnknown/1.0 bot", true},
		{"generic crawler", "MyCrawler/1.0", true},
		{"generic spider", "AwesomeSpider/3.2", true},

		// CLI fetchers — best-effort, but unsafe defaults are fine.
		{"curl", "curl/8.4.0", true},
		{"wget", "Wget/1.21", true},
		{"python requests", "python-requests/2.31.0", true},
		{"go default", "Go-http-client/1.1", true},
		{"node fetch", "node-fetch/1.0", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLinkPreviewBot(tc.ua); got != tc.want {
				t.Errorf("isLinkPreviewBot(%q) = %v, want %v", tc.ua, got, tc.want)
			}
		})
	}
}
