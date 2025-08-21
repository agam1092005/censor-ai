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
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gocv.io/x/gocv"
)

const (
	uploadFolder    = "uploads"
	processedFolder = "processed"
	maxFileSize     = 20 * 1024 * 1024 // 20MB
)

type RatingResult struct {
	Start  float64 `json:"start"`
	End    float64 `json:"end"`
	Rating string  `json:"rating"`
	Notes  string  `json:"notes"`
}

type ConvertRequest struct {
	Age       string         `json:"age" binding:"required,oneof=6 12 16"`
	Ratings   []RatingResult `json:"ratings" binding:"required"`
	VideoType string         `json:"video_type" binding:"required,oneof=blur trim"`
	VideoPath string         `json:"video_path" binding:"required"`
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
	os.MkdirAll(processedFolder, os.ModePerm)

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
	router.POST("/convert", convertVideo)
	router.GET("/download/:filename", downloadVideo)

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

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
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

	if len(openAIResp.Choices) == 0 {
		return "", "", fmt.Errorf("no choices in response")
	}

	content := openAIResp.Choices[0].Message.Content
	fmt.Println("Raw OpenAI Response:", content) // Debug print

	jsonStart := strings.Index(content, "{")
	if jsonStart == -1 {
		return "", "", fmt.Errorf("no JSON object found in response")
	}

	jsonText := content[jsonStart:]
	jsonText = strings.TrimSpace(jsonText)
	jsonText = strings.TrimSuffix(jsonText, "```")

	var ratingData RatingData
	if err := json.Unmarshal([]byte(jsonText), &ratingData); err != nil {
		return "", "", fmt.Errorf("failed to parse rating data: %v", err)
	}

	return ratingData.Rating, ratingData.Notes, nil
}

func convertVideo(c *gin.Context) {
	age := c.PostForm("age")
	videoType := c.PostForm("video_type")
	ratingsStr := c.PostForm("ratings")

	log.Printf("Raw ratings string: %s", ratingsStr)

	file, err := c.FormFile("video_path")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No video file provided"})
		return
	}

	var ratings []RatingResult

	if ratingsStr != "" {
		if err := json.Unmarshal([]byte(ratingsStr), &ratings); err != nil {
			log.Printf("Error parsing ratings: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid ratings format: %v", err)})
			return
		}
		log.Printf("Parsed %d rating segments", len(ratings))
		for i, r := range ratings {
			log.Printf("Rating %d: Start=%.2f, End=%.2f, Rating=%s, Notes=%s",
				i, r.Start, r.End, r.Rating, r.Notes)
		}
	}

	if age == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Age is required"})
		return
	}

	if videoType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Video type is required"})
		return
	}

	if age != "6" && age != "12" && age != "16" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Age must be one of: 6, 12, 16"})
		return
	}

	if videoType != "blur" && videoType != "trim" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Video type must be one of: blur, trim"})
		return
	}

	filename := filepath.Join(uploadFolder, file.Filename)
	if err := c.SaveUploadedFile(file, filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save video file"})
		return
	}

	log.Printf("Received convert request: Age=%s, VideoType=%s, VideoFile=%s", age, videoType, file.Filename)

	ageInt, err := strconv.Atoi(age)
	if err != nil {
		log.Printf("Error converting age '%s' to integer: %v", age, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid age format: %s", age)})
		os.Remove(filename)
		return
	}

	log.Printf("Converting age string '%s' to integer: %d", age, ageInt)

	outputPath, err := processVideoByAge(filename, ageInt, ratings, videoType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		os.Remove(filename)
		return
	}

	os.Remove(filename)

	baseFilename := filepath.Base(outputPath)

	host := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}

	downloadURL := fmt.Sprintf("%s://%s/download/%s", scheme, host, baseFilename)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Video processed successfully",
		"filename":     baseFilename,
		"download_url": downloadURL,
	})
}

func downloadVideo(c *gin.Context) {
	filename := c.Param("filename")
	filePath := filepath.Join(processedFolder, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "video/mp4")
	c.File(filePath)
}

func processVideoByAge(videoPath string, age int, ratings []RatingResult, videoType string) (string, error) {
	timestamp := time.Now().UnixNano()
	outputFilename := fmt.Sprintf("processed_%d.mp4", timestamp)
	outputPath := filepath.Join(processedFolder, outputFilename)

	video, err := gocv.VideoCaptureFile(videoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open video: %v", err)
	}
	defer video.Close()

	fps := video.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		fps = 30 // Default to 30fps if unable to determine
	}
	width := int(video.Get(gocv.VideoCaptureFrameWidth))
	height := int(video.Get(gocv.VideoCaptureFrameHeight))
	totalFrames := int(video.Get(gocv.VideoCaptureFrameCount))

	writer, err := gocv.VideoWriterFile(
		outputPath,
		"mp4v", // codec
		fps,
		width,
		height,
		true,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create video writer: %v", err)
	}
	defer writer.Close()

	if videoType == "blur" {
		err = blurInappropriateContent(video, writer, ratings, age, fps, totalFrames)
	} else {
		err = trimInappropriateContent(video, writer, ratings, age, fps, totalFrames) // trim
	}

	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func blurInappropriateContent(video *gocv.VideoCapture, writer *gocv.VideoWriter, ratings []RatingResult, age int, fps float64, totalFrames int) error {
	img := gocv.NewMat()
	defer img.Close()

	blurred := gocv.NewMat()
	defer blurred.Close()

	frameIndex := 0
	for {
		if ok := video.Read(&img); !ok || img.Empty() || frameIndex >= totalFrames {
			break
		}

		timestamp := float64(frameIndex) / fps
		shouldBlur := false

		for _, rating := range ratings {
			if timestamp >= rating.Start && timestamp <= rating.End {
				ratingValue := getRatingValue(rating.Rating)
				if ratingValue > age {
					shouldBlur = true
					break
				}
			}
		}

		if shouldBlur {
			gocv.GaussianBlur(img, &blurred, image.Point{X: 45, Y: 45}, 0, 0, gocv.BorderDefault)
			writer.Write(blurred)
		} else {
			writer.Write(img)
		}

		frameIndex++
	}

	return nil
}

func trimInappropriateContent(video *gocv.VideoCapture, writer *gocv.VideoWriter, ratings []RatingResult, age int, fps float64, totalFrames int) error {
	img := gocv.NewMat()
	defer img.Close()

	frameIndex := 0
	includedFrames := 0
	log.Printf("Starting trim process: Age=%d, FPS=%f, TotalFrames=%d", age, fps, totalFrames)
	log.Printf("Ratings data: %+v", ratings)

	// Sort ratings by start time to ensure we process them in order
	sort.Slice(ratings, func(i, j int) bool {
		return ratings[i].Start < ratings[j].Start
	})

	for {
		if ok := video.Read(&img); !ok || img.Empty() || frameIndex >= totalFrames {
			break
		}

		timestamp := float64(frameIndex) / fps
		shouldInclude := false // Default to not including the frame
		matchedRating := "none"
		inRatedSegment := false

		// Check if this frame is in any rated segment
		for _, rating := range ratings {
			// Use a small epsilon for floating point comparison to avoid rounding issues
			const epsilon = 0.001
			isAfterStart := timestamp >= (rating.Start - epsilon)
			isBeforeEnd := timestamp <= (rating.End + epsilon)

			if isAfterStart && isBeforeEnd {
				inRatedSegment = true
				matchedRating = rating.Rating

				// Only include if the rating is appropriate for the age
				ratingValue := getRatingValue(rating.Rating)
				if ratingValue <= age {
					shouldInclude = true
					break
				} else {
					// If we found a segment but it's not appropriate, don't include
					shouldInclude = false
					break
				}
			}
		}

		// If frame is not in any rated segment, don't include it
		if !inRatedSegment {
			matchedRating = "unrated"
			shouldInclude = false
		}

		if frameIndex%int(fps) == 0 { // Log once per second
			log.Printf("Frame %d (%.2fs): Rating=%s, InRatedSegment=%v, Include=%v",
				frameIndex, timestamp, matchedRating, inRatedSegment, shouldInclude)
		}

		if shouldInclude {
			writer.Write(img)
			includedFrames++
		}

		frameIndex++
	}

	log.Printf("Trim complete: Processed %d frames, Included %d frames (%.2f seconds)",
		frameIndex, includedFrames, float64(includedFrames)/fps)
	return nil
}

func getRatingValue(rating string) int {
	var value int
	switch rating {
	case "6+":
		value = 6
	case "12+":
		value = 12
	case "16+":
		value = 16
	case "18+":
		value = 18
	default:
		value = 0
	}
	log.Printf("Converting rating '%s' to value: %d", rating, value)
	return value
}
