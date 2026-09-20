package e2e

import (
	"bytes"
	"fmt"
)

// log collects everything printed about a single Test or Sequence run: the request/response
// trace, plus any warnings, errors, failures, or the final success line. Runner.Run holds on
// to it until every test has finished, then prints it (or not, depending on whether only
// failures were asked for).
type log struct {
	bytes.Buffer
}

// warning reports something that could plausibly be fine at runtime, e.g. a $-reference with
// nothing captured for it yet. Doesn't affect whether the test passes.
func (l *log) warning(format string, args ...any) {
	fmt.Fprintf(l, "%s: "+format+"\n", append([]any{yellow("WARNING")}, args...)...)
}

// error reports a problem that stops the test before it could run to completion, e.g. invalid
// setup caught by validate, or the request itself failing to execute. Wrap err first with
// fmt.Errorf and %w if it needs context, e.g. fmt.Errorf("making request: %w", err). The
// message starts on its own line so a multi-line error (errors.Join, for instance) stays
// aligned under it instead of trailing off after "ERROR: ".
func (l *log) error(err error) {
	fmt.Fprintf(l, "%s:\n%v\n", pink("ERROR"), err)
}

// fail reports an assertion that didn't hold, e.g. an unexpected status code. Wrap err first
// with fmt.Errorf and %w if it needs context, e.g. fmt.Errorf("status: %w", err).
func (l *log) fail(err error) {
	fmt.Fprintf(l, "%s: %v\n", pink("FAIL"), err)
}

// send reports one piece of the outgoing request, e.g. the request line, a single header, or
// the (possibly multi-line, pretty-printed) body, prefixed with a grey "->" arrow.
func (l *log) send(format string, args ...any) {
	fmt.Fprint(l, grey("->")+" "+ensureEndingNL(fmt.Sprintf(format, args...)))
}

// receive is send's counterpart for the incoming response, prefixed with a grey "<-" arrow.
func (l *log) receive(format string, args ...any) {
	fmt.Fprint(l, grey("<-")+" "+ensureEndingNL(fmt.Sprintf(format, args...)))
}

// success reports that every assertion held.
func (l *log) success() {
	fmt.Fprintln(l, green("SUCCESS"))
}

// banner prints name centered and dash-padded, in yellow, as the title line at the top of a
// test's output.
func (l *log) banner(name string) {
	fmt.Fprintln(l, yellow(center(name, 31)))
}

// separator prints a yellow dashed rule, followed by a blank line, marking the end of a
// Sequence's output.
func (l *log) separator() {
	fmt.Fprintln(l, yellow("---------------------------------\n"))
}

// print prints args the same way fmt.Fprintln would. Used for plain, uncolored lines that
// don't fit any of log's other, tagged methods.
func (l *log) print(args ...any) {
	fmt.Fprintln(l, args...)
}
