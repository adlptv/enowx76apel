package localcreds

import "github.com/enowdev/enowx/core/provider/kiro"

// sources is the registry of known local credential files. Add an entry here
// when a provider's IDE/CLI writes credentials to a predictable path.
var sources = []Source{
	{
		Provider: "kiro",
		Target:   "Kiro Desktop",
		rel:      []string{".aws", "sso", "cache", "kiro-auth-token.json"},
		parse:    kiro.NormalizeCreds,
	},
	{
		Provider: "kiro",
		Target:   "Kiro CLI",
		rel:      []string{".aws", "sso", "cache", "kiro-auth-token-cli.json"},
		parse:    kiro.NormalizeCreds,
	},
	// Future: codex (~/.codex/auth.json), opencode
	// (~/.config/opencode/opencode.json) — register once those providers exist.
}
