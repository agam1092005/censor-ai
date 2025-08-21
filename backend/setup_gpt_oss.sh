#!/bin/bash

# GPT-OSS Setup Script for Censor-AI
echo "Setting up GPT-OSS integration for Censor-AI..."

# Check if Python 3 is installed
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is required but not installed."
    echo "Please install Python 3.8 or higher."
    exit 1
fi

echo "Python 3 found: $(python3 --version)"

# Check if pip is installed
if ! command -v pip3 &> /dev/null; then
    echo "Error: pip3 is required but not installed."
    echo "Please install pip for Python 3."
    exit 1
fi

echo "Installing Python dependencies..."

# Create virtual environment (optional but recommended)
if [ "$1" == "--venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv gpt_oss_env
    source gpt_oss_env/bin/activate
    echo "Virtual environment activated."
fi

# Install dependencies
pip3 install -r requirements.txt

echo "Testing GPT-OSS classifier..."

# Test the classifier with sample data
test_input='{"metadata": {"filename": "test.mp4"}, "transcript": "Hello world", "vision_labels": ["general_content"]}'

if python3 oss_classifier.py "$test_input" > /dev/null 2>&1; then
    echo "✅ GPT-OSS classifier is working correctly!"
else
    echo "⚠️  GPT-OSS classifier test failed. Check your setup."
    echo "You may need to:"
    echo "1. Set OPENAI_API_KEY environment variable for OpenAI backend"
    echo "2. Or ensure transformers library is properly installed for offline mode"
fi

echo ""
echo "Setup complete! GPT-OSS integration is ready."
echo ""
echo "Usage:"
echo "1. Start the Go backend: go run main.go"
echo "2. Upload videos to /upload endpoint (includes GPT-OSS classification)"
echo "3. Use /classify endpoint for standalone content classification"
echo ""
echo "Environment variables:"
echo "- OPENAI_API_KEY: Set this to use OpenAI backend (recommended)"
echo "- If not set, will fallback to local Hugging Face models"
