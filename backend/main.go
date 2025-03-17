package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gocv.io/x/gocv"
)

const (
	uploadFolder = "uploads"
	maxFileSize  = 20 * 1024 * 1024 // 20MB
)

type RatingResult struct {
	Start  float64 `json:"start"`
	End    float64 `json:"end"`
	Rating string  `json:"rating"`
	Notes  string  `json:"notes"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type RatingData struct {
	Rating string `json:"rating"`
	Notes  string `json:"notes"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	os.MkdirAll(uploadFolder, os.ModePerm)

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"POST"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.MaxMultipartMemory = maxFileSize

	router.POST("/upload", uploadVideo)

	log.Println("Starting server on port 8000...")
	router.Run(":8000")
}

func uploadVideo(c *gin.Context) {
	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No video file provided"})
		return
	}

	filename := filepath.Join(uploadFolder, file.Filename)
	if err := c.SaveUploadedFile(file, filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video file"})
		return
	}

	ratings, err := processVideo(filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	os.Remove(filename)
	c.JSON(http.StatusOK, gin.H{"ratings": ratings})
}

func processVideo(videoPath string) ([]RatingResult, error) {
	// Open video file
	video, err := gocv.VideoCaptureFile(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open video: %v", err)
	}
	defer video.Close()

	fps := video.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 30 // Default to 30fps if unable to determine
	}

	var results []RatingResult
	frameIndex := 0

	var lastRating string
	var startTime float64
	combinedNotes := make(map[string]bool)

	img := gocv.NewMat()
	defer img.Close()

	for {
		if ok := video.Read(&img); !ok || img.Empty() {
			break
		}

		if frameIndex%int(fps) == 0 {
			timestamp := float64(frameIndex) / fps

			resized := gocv.NewMat()
			gocv.Resize(img, &resized, image.Point{X: 512, Y: 512}, 0, 0, gocv.InterpolationLinear)

			buf, err := gocv.IMEncode(".jpg", resized)
			resized.Close()
			if err != nil {
				frameIndex++
				continue
			}

			base64Img := base64.StdEncoding.EncodeToString(buf.GetBytes())
			dataURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64Img)
			rating, notes, err := analyzeFrameWithOpenAI(dataURL)
			if err != nil {
				return []RatingResult{{
					Start:  timestamp,
					Rating: fmt.Sprintf("Error: %v", err),
				}}, nil
			}

			if rating == lastRating {
				for _, note := range strings.Split(notes, ",") {
					note = strings.TrimSpace(strings.ToLower(note))
					if note != "" && note != "none" {
						combinedNotes[note] = true
					}
				}
			} else {
				if lastRating != "" {
					var notesList []string
					for note := range combinedNotes {
						notesList = append(notesList, note)
					}
					sort.Strings(notesList)
					notesStr := strings.Join(notesList, ", ")

					results = append(results, RatingResult{
						Start:  startTime,
						End:    timestamp - 1,
						Rating: lastRating,
						Notes:  notesStr,
					})
				}

				startTime = timestamp
				lastRating = rating
				combinedNotes = make(map[string]bool)
				for _, note := range strings.Split(notes, ",") {
					note = strings.TrimSpace(strings.ToLower(note))
					if note != "" && note != "none" {
						combinedNotes[note] = true
					}
				}
			}
		}

		frameIndex++
	}

	if lastRating != "" {
		var notesList []string
		for note := range combinedNotes {
			notesList = append(notesList, note)
		}
		sort.Strings(notesList)
		notesStr := strings.Join(notesList, ", ")

		results = append(results, RatingResult{
			Start:  startTime,
			End:    float64(frameIndex) / fps,
			Rating: lastRating,
			Notes:  notesStr,
		})
	}

	return results, nil
}

func analyzeFrameWithOpenAI(dataURL string) (string, string, error) {
	type Message struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"`
	}

	type ImageURL struct {
		URL string `json:"url"`
	}

	type ContentItem struct {
		Type     string    `json:"type"`
		Text     string    `json:"text,omitempty"`
		ImageURL *ImageURL `json:"image_url,omitempty"`
	}

	promptText := `Analyze the image and rate it for violence and nudity based on these guidelines:

- **6+**: Minimal, non-detailed violence. No nudity.
- **12+**: Moderate violence without injury detail. Brief, non-sexual nudity.
- **16+**: Intense but non-gratuitous violence. Partial nudity and implied sexual content allowed.
- **18+**: Explicit violence with gore. Nudity, including sexual content, allowed.

Return a valid JSON object with two fields:
{
  "rating": "one of 18+, 16+, 12+, 6+",
  "notes": "comma-separated keywords describing content (e.g. 'blood, nude')"
}`

	contentItems := []ContentItem{
		{
			Type: "image_url",
			ImageURL: &ImageURL{
				URL: dataURL,
			},
		},
		{
			Type: "text",
			Text: promptText,
		},
	}

	requestBody := map[string]interface{}{
		"model": "gpt-4o",
		"messages": []Message{
			{
				Role:    "user",
				Content: contentItems,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %v", err)
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %v", err)
	}

	return openAIResp.Choices[0].Message.Content, "", nil
}
