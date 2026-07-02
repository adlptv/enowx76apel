// Package sanitize rewrites content on the way to an upstream provider and
// reverses it on the way back. Some providers block certain words (brand names,
// etc.); a rule swaps the blocked word for a safe stand-in before sending
// (Obfuscate) and restores the original in the reply (Deobfuscate).
package sanitize

import (
	"regexp"
	"strings"
	"sync"
)

// Rule is one pattern→replacement mapping (optionally a regular expression).
type Rule struct {
	Pattern     string
	Replacement string
	Regex       bool
}

// LooksRegex reports whether a pattern should be treated as a regular expression:
// it contains a regex metacharacter AND compiles. Plain text (e.g. "enowx") stays
// literal; a pattern like "foo.*bar" or "co(de|re)" is detected as regex. Invalid
// regex falls back to literal (returns false) so a bad pattern never breaks.
func LooksRegex(pattern string) bool {
	if !strings.ContainsAny(pattern, `.*+?()[]{}|^$\`) {
		return false
	}
	_, err := regexp.Compile(pattern)
	return err == nil
}

type compiled struct {
	pattern     string
	replacement string
	re          *regexp.Regexp
}

var (
	mu      sync.RWMutex
	forward []compiled // pattern → replacement (applied upstream)
	reverse []compiled // replacement → pattern (applied downstream)
)

// SetRules installs the active rule set (called on boot + on every change).
func SetRules(rules []Rule) {
	fwd := make([]compiled, 0, len(rules))
	rev := make([]compiled, 0, len(rules))
	for _, r := range rules {
		if r.Pattern == "" || r.Pattern == r.Replacement {
			continue
		}
		c := compiled{pattern: r.Pattern, replacement: r.Replacement}
		if r.Regex {
			re, err := regexp.Compile(r.Pattern)
			if err != nil {
				continue // skip invalid regex rather than break everything
			}
			c.re = re
		}
		fwd = append(fwd, c)

		// Reverse map: skip identical / empty / too-short replacements, which
		// would match too broadly and corrupt unrelated text.
		if r.Replacement == "" || len(r.Replacement) < 3 {
			continue
		}
		rc := compiled{pattern: r.Replacement, replacement: r.Pattern}
		// A regex forward rule has no safe literal reverse, so only reverse
		// literal rules.
		if !r.Regex {
			rev = append(rev, rc)
		}
	}
	mu.Lock()
	forward, reverse = fwd, rev
	mu.Unlock()
}

// Active reports whether any rules are installed (fast no-op guard).
func Active() bool {
	mu.RLock()
	defer mu.RUnlock()
	return len(forward) > 0
}

// Obfuscate rewrites request text (pattern → replacement).
func Obfuscate(s string) string {
	mu.RLock()
	rules := forward
	mu.RUnlock()
	return apply(s, rules)
}

// Deobfuscate restores response text (replacement → pattern).
func Deobfuscate(s string) string {
	mu.RLock()
	rules := reverse
	mu.RUnlock()
	return apply(s, rules)
}

func apply(s string, rules []compiled) string {
	for _, r := range rules {
		if r.re != nil {
			s = r.re.ReplaceAllString(s, r.replacement)
		} else {
			s = strings.ReplaceAll(s, r.pattern, r.replacement)
		}
	}
	return s
}
