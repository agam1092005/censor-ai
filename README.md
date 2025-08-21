# CENSOR AI

A Go backend with a Next.js frontend that leverages OpenAI to censor videos based on age categories (6+, 12+, 16+, 18+). Supports blurring, trimming, and muting inappropriate content based on the selected age filter.

## DEMO Video

https://www.youtube.com/watch?v=epCRTlCXfpU&t=121s

## Prerequisites

Before testing the application, ensure you have the following installed:

- **Go 1.23.3 or higher** - [Download Go](https://golang.org/dl/)
- **Node.js 18.0 or higher** - [Download Node.js](https://nodejs.org/)
- **npm or yarn** - Comes with Node.js
- **OpenCV** - Required for video processing
- **FFmpeg** - Required for video manipulation

### Installing OpenCV (Required for Backend)

#### macOS

```bash
brew install opencv
```

#### Ubuntu/Debian

```bash
sudo apt update
sudo apt install libopencv-dev
```

#### Windows

Follow the [OpenCV installation guide](https://docs.opencv.org/4.x/d3/d52/tutorial_windows_install.html)

### Installing FFmpeg (Required for Video Processing)

#### macOS

```bash
brew install ffmpeg
```

#### Ubuntu/Debian

```bash
sudo apt update
sudo apt install ffmpeg
```

#### Windows

Download from [FFmpeg official website](https://ffmpeg.org/download.html)

## Setup Instructions

### 1. Clone the Repository

```bash
git clone https://github.com/agam1092005/censor-ai.git
cd censor-ai
```

### 2. Backend Setup

#### Step 1: Navigate to Backend Directory

```bash
cd backend
```

#### Step 2: Create Environment File

Create a `.env` file in the backend directory with your OpenAI API key:

```bash
echo "OPENAI_API_KEY=your_openai_api_key_here" > .env
```

**Important:** Replace `your_openai_api_key_here` with your actual OpenAI API key from [OpenAI Platform](https://platform.openai.com/api-keys)

#### Step 3: Install Go Dependencies

```bash
go mod download
```

#### Step 4: Run Backend Server

```bash
go run main.go
```

The backend server will start on `http://localhost:8000`

You should see:

```
Starting server on port 8000...
[GIN] Listening on :8000
```

### 3. Frontend Setup

#### Step 1: Open New Terminal and Navigate to Frontend

```bash
cd frontend
```

#### Step 2: Install Dependencies

```bash
npm install
# or
yarn install
```

#### Step 3: Run Frontend Development Server

```bash
npm run dev
# or
yarn dev
```

The frontend will start on `http://localhost:3000`

## Testing Instructions

### 1. Access the Application

Open your web browser and navigate to `http://localhost:3000`

### 2. Prepare Test Videos

#### Sample Test Videos

For testing purposes, you can use any of the following types of videos:

- **Family-friendly content** (should get 6+ rating)
- **Mild content** (should get 12+ rating)
- **Moderate content** (should get 16+ rating)
- **Mature content** (should get 18+ rating)

#### Video Requirements

- **Format:** MP4, AVI, MOV, or other common video formats
- **Size:** Maximum 20MB
- **Duration:** Shorter videos (1-5 minutes) work best for testing
- **Content:** Videos with varying content levels to test different age ratings

#### Where to Find Test Videos

- Use your own video files
- Download sample videos from:
  - [Sample Videos](https://sample-videos.com/)
  - [Big Buck Bunny](https://peach.blender.org/download/) (family-friendly)
  - Create short screen recordings for testing

### 3. Testing the Video Upload and Processing

#### Step 1: Upload Video

1. On the main page, you'll see a drag-and-drop area
2. Either:
   - **Drag and drop** a video file onto the upload area
   - **Click** the upload area to select a file from your computer

#### Step 2: Analyze Video

1. After uploading, click the **"Analyze Video"** button
2. The application will:
   - Upload the video to the backend
   - Process the video frame by frame
   - Use OpenAI to analyze content appropriateness
   - Return age ratings for different segments

#### Step 3: Select Age Rating and Processing Option

1. **Choose Age Rating:** Select from 6+, 12+, or 16+
2. **Choose Processing Method:**
   - **Blur:** Blurs inappropriate sections while keeping audio
   - **Trim:** Removes inappropriate sections entirely (auto-selected for 6+)

#### Step 4: Process Video

1. Click **"Process Video"**
2. Wait for processing to complete
3. Download the processed video

### 4. Expected Results

#### Processing Times

- **Small videos (< 1MB):** 30 seconds - 2 minutes
- **Medium videos (1-5MB):** 2-5 minutes
- **Large videos (5-20MB):** 5-15 minutes

#### Output Location

Processed videos are saved in `backend/processed/` with timestamps

#### Success Indicators

- ✅ Video uploads successfully
- ✅ Analysis completes without errors
- ✅ Age ratings are returned
- ✅ Processing completes successfully
- ✅ Processed video can be downloaded

### 5. Troubleshooting

#### Common Issues

**Backend won't start:**

- Check if OpenCV is properly installed
- Verify Go version (1.23.3+)
- Ensure port 8000 is not in use

**Frontend won't start:**

- Check Node.js version (18.0+)
- Run `npm install` again
- Ensure port 3000 is not in use

**Video upload fails:**

- Check file size (max 20MB)
- Verify video format is supported
- Ensure both backend and frontend are running

**OpenAI API errors:**

- Verify API key is correct in `.env` file
- Check OpenAI account has credits
- Ensure internet connection is stable

**Video processing fails:**

- Check FFmpeg is installed and in PATH
- Verify enough disk space for processing
- Check video file is not corrupted

#### API Endpoints for Manual Testing

You can also test the backend directly:

**Upload endpoint:**

```bash
curl -X POST -F "video=@/path/to/your/video.mp4" http://localhost:8000/upload
```

**Convert endpoint:**

```bash
curl -X POST http://localhost:8000/convert \
  -H "Content-Type: application/json" \
  -d '{
    "age": "12",
    "ratings": [{"start": 0, "end": 10, "rating": "12+", "notes": "mild content"}],
    "video_type": "blur",
    "video_path": "uploads/your_video.mp4"
  }'
```

### 6. Sample Testing Workflow

1. **Start both servers** (backend on :8000, frontend on :3000)
2. **Upload a test video** (try a 1-2 minute video first)
3. **Analyze the video** and wait for ratings
4. **Select age rating** (try 12+ first)
5. **Choose processing type** (try "blur" first)
6. **Process the video** and download result
7. **Verify the output** by playing the processed video

## GPT-OSS Content Classification

The project includes GPT-OSS integration for intelligent content classification that provides overall content ratings based on video metadata, audio transcripts, and vision analysis.

### Setup GPT-OSS

#### Step 1: Install Python Dependencies

```bash
cd backend
./setup_gpt_oss.sh
```

Or manually:

```bash
pip3 install -r requirements.txt
```

#### Step 2: Configure Backend (Optional)

For best results, set your OpenAI API key:

```bash
export OPENAI_API_KEY="your-openai-api-key"
```

If no API key is provided, the system automatically uses offline Hugging Face models.

### Testing GPT-OSS Integration

#### Automated Test Suite

```bash
cd backend
python3 test_gpt_oss.py
```

This will test:

- Direct Python classifier functionality
- Go backend integration
- Different content type classifications

#### Manual API Testing

**Standalone Classification:**

```bash
curl -X POST http://localhost:8000/classify \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {"filename": "test.mp4", "duration": 120},
    "transcript": "Family-friendly adventure story with no violence",
    "vision_labels": ["adventure", "family", "outdoor"]
  }'
```

**Enhanced Video Upload:**

```bash
curl -X POST http://localhost:8000/upload -F "video=@your_video.mp4"
```

The response now includes both frame-by-frame analysis and overall GPT-OSS classification:

```json
{
  "ratings": [...],
  "gpt_oss": {
    "rating": "12+",
    "reason": "Moderate action content suitable for ages 12 and above"
  }
}
```

#### Demo Script

Run the comprehensive demo:

```bash
cd backend
./demo_gpt_oss.sh
```

This demonstrates classification with different content types and shows the complete integration workflow.

### GPT-OSS Features

- **Dual Backend Support**: OpenAI API (online) or Hugging Face (offline)
- **Intelligent Classification**: Analyzes metadata, transcripts, and vision labels
- **Age Ratings**: Returns 6+, 12+, 16+, or 18+ with explanations
- **Automatic Integration**: Works alongside existing video processing
- **Fallback Support**: Graceful degradation if classification fails

### Troubleshooting GPT-OSS

**Import Errors:**

```bash
pip3 install openai transformers torch
```

**Model Loading Issues:**

- First-time Hugging Face model download may take 5-10 minutes
- Models automatically fallback to CPU if CUDA unavailable
- Check available disk space (models require 1-5GB)

**API Errors:**

- Verify `OPENAI_API_KEY` is correctly set
- Check OpenAI account credits and rate limits
- System will automatically fallback to offline models
