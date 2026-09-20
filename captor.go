package e2e

import (
	"fmt"
	"net/http"
	"regexp"
)

// A Captor captures a value from an HTTP response for later reference via the $-prefix.
type Captor struct {
	// Name is the response body field or header to capture, using the same dot-separated
	// path syntax as Expect.Body for body fields.
	Name string
	// As is the key the captured value is stored under. If empty, Name is used instead.
	// Reference the stored value elsewhere using the $-prefix, e.g. $Name or $As.
	As string
	// Regex, if set, cherry-picks part of the captured value instead of capturing it in
	// full: the pattern's first capturing group is captured if it has one, otherwise the
	// whole match is. Use non-capturing groups, (?:...), for whichever part of the pattern
	// isn't what should be captured.
	Regex string
}

// key is the key the captured value should be stored under: As if set, otherwise Name.
func (c Captor) key() string {
	if c.As != "" {
		return c.As
	}
	return c.Name
}

// capture finds c's value in body or headers (body checked first, headers only if nothing
// matched there), applies Regex if set, and returns it. ok is false, with a warning already
// logged, if nothing matched at all, or Regex was invalid or matched nothing.
func (c Captor) capture(body map[string][]string, headers http.Header, log *log) (value string, ok bool) {
	val, foundMatch := body[c.Name]
	if !foundMatch {
		val, foundMatch = headers[http.CanonicalHeaderKey(c.Name)]
	}
	if !foundMatch {
		log.warning("capturing %q: matched nothing.", c.Name)
		return "", false
	}
	if len(val) > 1 {
		log.warning("capturing %q: matched multiple values. Captures first one.", c.Name)
	}

	value = fmt.Sprint(val[0])
	if c.Regex == "" {
		return value, true
	}
	return c.extract(value, log)
}

// extract cherry-picks part of value using c.Regex: the pattern's first capturing group if it
// has one, otherwise the whole match. ok is false, with a warning already logged, if Regex is
// invalid or matches nothing.
func (c Captor) extract(value string, log *log) (result string, ok bool) {
	re, err := regexp.Compile(c.Regex)
	if err != nil { // should be a real error checked at validation. Compile regex at validation an store in a non-exported field to be used later.
		log.warning("capturing %q: invalid Regex %q: %v", c.Name, c.Regex, err)
		return "", false
	}
	match := re.FindStringSubmatch(value)
	if match == nil {
		log.warning("capturing %q: regex %q matched nothing in the captured value.", c.Name, c.Regex)
		return "", false
	}
	// match[1] is the first capturing group, if the pattern has one; otherwise match[0], the
	// whole match, is all there is.
	if len(match) > 1 {
		return match[1], true
	}
	return match[0], true
}
