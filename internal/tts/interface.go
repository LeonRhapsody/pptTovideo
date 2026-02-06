package tts

// TTSProvider defines the interface for Text-to-Speech synthesis.
type TTSProvider interface {
	// Synthesize converts text to speech and saves it to the specified output path.
	Synthesize(text string, outputPath string, voiceName string) error
}
