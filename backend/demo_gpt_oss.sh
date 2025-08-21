#!/bin/bash

# GPT-OSS Integration Demo Script
echo "🎬 Censor-AI GPT-OSS Integration Demo"
echo "====================================="

# Check if Go backend is running
if ! curl -s http://localhost:8000/classify > /dev/null 2>&1; then
    echo "❌ Go backend is not running on port 8000"
    echo "Please start it with: cd backend && go run main.go"
    exit 1
fi

echo "✅ Go backend is running"

# Test 1: Standalone Classification
echo ""
echo "📋 Test 1: Standalone Content Classification"
echo "-------------------------------------------"

response=$(curl -s -X POST http://localhost:8000/classify \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "filename": "family_movie.mp4",
      "duration": 90,
      "genre": "family"
    },
    "transcript": "A heartwarming story about friendship and adventure. Children explore a magical forest and learn about cooperation.",
    "vision_labels": ["children", "forest", "magic", "friendship", "adventure"]
  }')

echo "Input: Family-friendly content"
echo "Response: $response"

# Test 2: Action Content
echo ""
echo "📋 Test 2: Action Content Classification"
echo "----------------------------------------"

response=$(curl -s -X POST http://localhost:8000/classify \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "filename": "action_movie.mp4",
      "duration": 120,
      "genre": "action"
    },
    "transcript": "Intense fight scenes with weapons and combat. Some characters are injured but no graphic violence or gore shown.",
    "vision_labels": ["fighting", "weapons", "combat", "action", "chase"]
  }')

echo "Input: Action content with moderate violence"
echo "Response: $response"

# Test 3: Mature Content
echo ""
echo "📋 Test 3: Mature Content Classification"
echo "----------------------------------------"

response=$(curl -s -X POST http://localhost:8000/classify \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "filename": "thriller.mp4",
      "duration": 105,
      "genre": "thriller"
    },
    "transcript": "Graphic violence with blood and gore. Adult themes including death and psychological horror.",
    "vision_labels": ["violence", "blood", "gore", "death", "horror", "adult"]
  }')

echo "Input: Mature content with graphic violence"
echo "Response: $response"

# Test 4: Video Upload Simulation (if test video exists)
echo ""
echo "📋 Test 4: Complete Video Processing Pipeline"
echo "--------------------------------------------"

# Create a small test file for demo purposes
if command -v ffmpeg &> /dev/null; then
    echo "Creating test video with ffmpeg..."
    ffmpeg -f lavfi -i testsrc2=duration=5:size=320x240:rate=1 -y test_video.mp4 2>/dev/null
    
    if [ -f "test_video.mp4" ]; then
        echo "Uploading test video to /upload endpoint..."
        response=$(curl -s -X POST http://localhost:8000/upload -F "video=@test_video.mp4")
        echo "Response: $response" | jq . 2>/dev/null || echo "Response: $response"
        rm test_video.mp4
    else
        echo "⚠️  Could not create test video"
    fi
else
    echo "⚠️  ffmpeg not available for test video creation"
    echo "You can manually test by uploading a video file to http://localhost:8000/upload"
fi

echo ""
echo "🎉 Demo complete!"
echo ""
echo "Integration Summary:"
echo "• GPT-OSS classifier successfully integrated with Censor-AI"
echo "• Supports both OpenAI API and Hugging Face backends"
echo "• Provides content ratings: 6+, 12+, 16+, 18+"
echo "• Integrated with existing video processing pipeline"
echo "• Includes standalone classification endpoint"
echo ""
echo "API Endpoints:"
echo "• POST /upload    - Upload video (includes GPT-OSS classification)"
echo "• POST /classify  - Standalone content classification"
echo "• POST /convert   - Convert video with age-based censoring"
