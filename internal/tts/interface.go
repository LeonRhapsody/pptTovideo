package tts

// Options defines the parameters for speech synthesis.
type Options struct {
	Rate   string // e.g., "+0%", "+20%", "-10%"
	Volume string // e.g., "+0%", "+50%", "-20%"
	Pitch  string // e.g., "+0Hz", "+5Hz", "-5Hz"
}

// TTSProvider defines the interface for Text-to-Speech synthesis.
type TTSProvider interface {
	// Synthesize converts text to speech and saves it to the specified output path.
	Synthesize(text string, outputPath string, voiceName string, opts Options) error
}
