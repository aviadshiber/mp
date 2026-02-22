// Package iostreams provides an abstraction over standard I/O with TTY detection,
// color support, and quiet-mode filtering. It follows the gh-cli IOStreams pattern.
package iostreams

import (
	"fmt"
	"io"
	"os"

	"github.com/muesli/termenv"
)

// IOStreams bundles the three standard streams together with display options.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	quiet        bool
	colorEnabled bool
	profile      termenv.Profile
}

// New returns IOStreams wired to the real stdin/stdout/stderr.
// Color is enabled when stdout is a TTY and the NO_COLOR env var is not set.
func New() *IOStreams {
	profile := termenv.ColorProfile()
	colorEnabled := fileIsTerminal(os.Stdout) && os.Getenv("NO_COLOR") == ""

	return &IOStreams{
		In:           os.Stdin,
		Out:          os.Stdout,
		ErrOut:       os.Stderr,
		colorEnabled: colorEnabled,
		profile:      profile,
		quiet:        false,
	}
}

// SetQuiet enables or disables quiet mode. In quiet mode Printf is suppressed.
func (s *IOStreams) SetQuiet(q bool) {
	s.quiet = q
}

// IsQuiet reports whether quiet mode is active.
func (s *IOStreams) IsQuiet() bool {
	return s.quiet
}

// IsTerminal reports whether stdout is connected to a terminal.
func (s *IOStreams) IsTerminal() bool {
	if f, ok := s.Out.(*os.File); ok {
		return fileIsTerminal(f)
	}
	return false
}

// ColorEnabled reports whether colored output should be produced.
func (s *IOStreams) ColorEnabled() bool {
	return s.colorEnabled
}

// Printf writes formatted output to Out, suppressed in quiet mode.
func (s *IOStreams) Printf(format string, a ...any) {
	if s.quiet {
		return
	}
	fmt.Fprintf(s.Out, format, a...)
}

// Println writes a line to Out, suppressed in quiet mode.
func (s *IOStreams) Println(a ...any) {
	if s.quiet {
		return
	}
	fmt.Fprintln(s.Out, a...)
}

// Errorf writes formatted output to ErrOut. It is never suppressed.
func (s *IOStreams) Errorf(format string, a ...any) {
	fmt.Fprintf(s.ErrOut, format, a...)
}

// --- Color helpers ---

// Success returns text styled as green (success).
func (s *IOStreams) Success(text string) string {
	if !s.colorEnabled {
		return text
	}
	return termenv.String(text).Foreground(s.profile.Color("2")).String()
}

// Failure returns text styled as red (error).
func (s *IOStreams) Failure(text string) string {
	if !s.colorEnabled {
		return text
	}
	return termenv.String(text).Foreground(s.profile.Color("1")).String()
}

// Warning returns text styled as yellow.
func (s *IOStreams) Warning(text string) string {
	if !s.colorEnabled {
		return text
	}
	return termenv.String(text).Foreground(s.profile.Color("3")).String()
}

// Muted returns text styled as gray/dim.
func (s *IOStreams) Muted(text string) string {
	if !s.colorEnabled {
		return text
	}
	return termenv.String(text).Faint().String()
}

// Bold returns text styled as bold.
func (s *IOStreams) Bold(text string) string {
	if !s.colorEnabled {
		return text
	}
	return termenv.String(text).Bold().String()
}

// fileIsTerminal checks if f is a character device (terminal).
func fileIsTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
