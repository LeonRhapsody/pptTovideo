package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LeonRhapsody/pptTovideo/internal/config"
	"github.com/LeonRhapsody/pptTovideo/internal/ppt"
	"github.com/LeonRhapsody/pptTovideo/internal/tts"
	"github.com/LeonRhapsody/pptTovideo/internal/video"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Config *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{Config: cfg}
}

// -- Request/Response Structs --

type SlideData struct {
	Index    int    `json:"index"`
	ImageURL string `json:"image_url"` // Relative URL
	Text     string `json:"text"`
}

type ParseResponse struct {
	JobID  string      `json:"job_id"`
	Slides []SlideData `json:"slides"`
}

type PreviewRequest struct {
	Text       string `json:"text" binding:"required"`
	EngineType string `json:"engine_type" binding:"required"`
	VoiceName  string `json:"voice_name"`
	Rate       string `json:"rate"`
	Volume     string `json:"volume"`
	Pitch      string `json:"pitch"`
}

type RenderRequest struct {
	JobID            string      `json:"job_id" binding:"required"`
	EngineType       string      `json:"engine_type" binding:"required"`
	VoiceName        string      `json:"voice_name"`
	Rate             string      `json:"rate"`
	Volume           string      `json:"volume"`
	Pitch            string      `json:"pitch"`
	Slides           []SlideData `json:"slides" binding:"required"`
	EnableSubtitles  bool        `json:"enable_subtitles"`
	SubtitleFontSize int         `json:"subtitle_font_size"`
}

// splitTextIntoSentences splits text based on punctuation.
func splitTextIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for _, r := range runes {
		current.WriteRune(r)
		// Check delimiters: Chinese and English punctuation
		if strings.ContainsRune("。！？.!?\n", r) {
			s := strings.TrimSpace(current.String())
			if len(s) > 0 {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}

	s := strings.TrimSpace(current.String())
	if len(s) > 0 {
		sentences = append(sentences, s)
	}

	return sentences
}

// -- Handlers --

func (h *Handler) RenderIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func (h *Handler) HandleParse(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	jobID := fmt.Sprintf("%d", time.Now().UnixNano())
	workDir := filepath.Join("uploads", jobID)
	os.MkdirAll(workDir, 0755)

	pptxPath := filepath.Join(workDir, file.Filename)
	if err := c.SaveUploadedFile(file, pptxPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	errChan := make(chan error, 2)
	var slides []ppt.Slide
	var images []string
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var err error
		slides, err = ppt.ParsePPT(pptxPath)
		if err != nil {
			errChan <- fmt.Errorf("ppt parsing failed: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		imgDir := filepath.Join(workDir, "images")
		var err error
		images, err = ppt.ConvertSlidesToImages(pptxPath, imgDir)
		if err != nil {
			errChan <- fmt.Errorf("image conversion failed: %w", err)
		}
	}()

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		err := <-errChan
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responseSlides := make([]SlideData, 0)
	count := len(slides)
	if len(images) < count {
		count = len(images)
	}

	for i := 0; i < count; i++ {
		filename := filepath.Base(images[i])
		url := fmt.Sprintf("/uploads/%s/images/%s", jobID, filename)

		responseSlides = append(responseSlides, SlideData{
			Index:    i,
			ImageURL: url,
			Text:     slides[i].Note,
		})
	}

	c.JSON(http.StatusOK, ParseResponse{
		JobID:  jobID,
		Slides: responseSlides,
	})
}

func (h *Handler) HandlePreview(c *gin.Context) {
	var req PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider, err := tts.NewTTSProvider(tts.EngineType(req.EngineType), h.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid engine"})
		return
	}

	tmpFile, err := ioutil.TempFile("", "preview-*.mp3")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Temp file error"})
		return
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	opts := tts.Options{
		Rate:   req.Rate,
		Volume: req.Volume,
		Pitch:  req.Pitch,
	}

	processedText := req.Text
	// Replace [停顿] with something that causes a pause.
	// For Edge TTS, wrapping in SSML is best, but a simpler way for now
	// is to use periods which its natural processing understands.
	// However, we'll use a more explicit approach if needed.
	processedText = strings.ReplaceAll(processedText, "[停顿]", "... ")

	if err := provider.Synthesize(processedText, tmpFile.Name(), req.VoiceName, opts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.File(tmpFile.Name())
}

func (h *Handler) HandleGetConfig(c *gin.Context) {
	// Return config for frontend display.
	// Security note: We might want to mask secrets in a real prod app,
	// but for a local tool returning them is fine for editing.
	c.JSON(http.StatusOK, h.Config)
}

func (h *Handler) HandleSaveConfig(c *gin.Context) {
	var newCfg config.Config
	if err := c.ShouldBindJSON(&newCfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update current config
	// We need to be careful not to overwrite Port if it's not in JSON,
	// but frontend should send everything.
	// For simplicity, we just copy the fields we care about or replace the struct values

	h.Config.XunfeiAppID = newCfg.XunfeiAppID
	h.Config.XunfeiAPIKey = newCfg.XunfeiAPIKey
	h.Config.XunfeiAPISecret = newCfg.XunfeiAPISecret

	h.Config.VolcAccessKey = newCfg.VolcAccessKey
	h.Config.VolcSecretKey = newCfg.VolcSecretKey
	h.Config.VolcAppKey = newCfg.VolcAppKey

	h.Config.GoogleAPIKey = newCfg.GoogleAPIKey

	h.Config.OpenAIAPIKey = newCfg.OpenAIAPIKey
	h.Config.OpenAIBaseURL = newCfg.OpenAIBaseURL

	if err := h.Config.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration saved"})
}

func (h *Handler) HandleGetTasks(c *gin.Context) {
	jobs := GlobalJobManager.GetAllJobs()
	c.JSON(http.StatusOK, gin.H{"tasks": jobs})
}

func (h *Handler) HandleRender(c *gin.Context) {
	var req RenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SubtitleFontSize <= 0 {
		req.SubtitleFontSize = 48
	}

	workDir := filepath.Join("uploads", req.JobID)
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job expired or not found"})
		return
	}

	jobID := req.JobID
	GlobalJobManager.CreateJob(jobID)

	c.JSON(http.StatusOK, gin.H{"job_id": jobID, "message": "Rendering started"})

	go func() {
		GlobalJobManager.UpdateProgress(jobID, 10, "Initializing...")

		provider, err := tts.NewTTSProvider(tts.EngineType(req.EngineType), h.Config)
		if err != nil {
			GlobalJobManager.FailJob(jobID, "Invalid TTS engine")
			return
		}

		audioDir := filepath.Join(workDir, "audio_render")
		os.MkdirAll(audioDir, 0755)

		type Segment struct {
			Text      string
			AudioPath string
			ImagePath string
		}
		var segments []Segment

		for _, slide := range req.Slides {
			urlParts := strings.Split(slide.ImageURL, "/")
			filename := urlParts[len(urlParts)-1]
			imgPath := filepath.Join(workDir, "images", filename)

			rawText := slide.Text
			if len(strings.TrimSpace(rawText)) == 0 {
				segments = append(segments, Segment{
					Text:      "",
					ImagePath: imgPath,
				})
			} else {
				subSentences := splitTextIntoSentences(rawText)
				if len(subSentences) == 0 {
					subSentences = []string{rawText}
				}
				for _, s := range subSentences {
					segments = append(segments, Segment{
						Text:      s,
						ImagePath: imgPath,
					})
				}
			}
		}
		totalSegments := len(segments)

		for i, seg := range segments {
			progress := 10 + int(float64(i)/float64(totalSegments)*70.0)
			GlobalJobManager.UpdateProgress(jobID, progress, fmt.Sprintf("Synthesizing audio %d/%d", i+1, totalSegments))

			outPath := filepath.Join(audioDir, fmt.Sprintf("audio_%d.mp3", i))

			textToSpeak := seg.Text
			if len(strings.TrimSpace(textToSpeak)) == 0 {
				textToSpeak = "..."
			}
			textToSpeak = strings.ReplaceAll(textToSpeak, "[停顿]", "... ")

			opts := tts.Options{
				Rate:   req.Rate,
				Volume: req.Volume,
				Pitch:  req.Pitch,
			}

			if err := provider.Synthesize(textToSpeak, outPath, req.VoiceName, opts); err != nil {
				errMsg := fmt.Sprintf("TTS failed for segment %d: %v", i+1, err)
				GlobalJobManager.FailJob(jobID, errMsg)
				return
			}

			segments[i].AudioPath = outPath
		}

		var imagePaths []string
		var audioPaths []string
		var texts []string
		for _, seg := range segments {
			imagePaths = append(imagePaths, seg.ImagePath)
			audioPaths = append(audioPaths, seg.AudioPath)
			texts = append(texts, seg.Text)
		}

		GlobalJobManager.UpdateProgress(jobID, 85, "Rendering Video...")
		outputVideoPath := filepath.Join(workDir, fmt.Sprintf("output_%d.mp4", time.Now().Unix()))

		opts := video.RenderOptions{
			EnableSubtitles: req.EnableSubtitles,
			FontSize:        req.SubtitleFontSize,
		}

		if err := video.ComposeVideo(imagePaths, audioPaths, texts, outputVideoPath, opts); err != nil {
			errMsg := fmt.Sprintf("Video composition failed: %v", err)
			GlobalJobManager.FailJob(jobID, errMsg)
			return
		}

		downloadURL := fmt.Sprintf("/uploads/%s/%s", req.JobID, filepath.Base(outputVideoPath))
		GlobalJobManager.CompleteJob(jobID, downloadURL)
	}()
}
