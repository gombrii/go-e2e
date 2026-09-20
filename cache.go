package e2e

import (
	"regexp"
	"strings"
)

// Matches strings such as "$user", "$var.field.subfield", and "$X-Amzn-Trace-Id".
var refMatcher *regexp.Regexp = regexp.MustCompile(`\$[\w-]+(?:\.[\w-]+)*`)

// cache holds values captured from earlier steps (see Captor), made available to later ones
// via the $-prefix.
type cache map[string]string

// expand replaces every $-reference in s with its value from the cache, warning and leaving
// the reference as-is wherever nothing was captured for it. location names where s came from,
// for the warning message, e.g. "URL" or `header %q`.
func (c cache) expand(s, location string, log *log) string {
	return refMatcher.ReplaceAllStringFunc(s, func(m string) string {
		key := strings.TrimPrefix(m, "$")
		val, ok := c[key]
		if !ok {
			log.warning("%q in %s has no captured value, leaving reference as is", m, location)
			return m
		}
		return val
	})
}
