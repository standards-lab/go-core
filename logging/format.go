package logging

// Format selects the slog handler [New] constructs.
type Format string

// The supported output formats.
const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// String returns the format's string value.
func (f Format) String() string {
	return string(f)
}

// Valid reports whether the format is one New recognizes.
func (f Format) Valid() bool {
	switch f {
	case FormatText, FormatJSON:
		return true
	default:
		return false
	}
}
