#!/usr/bin/env python3
"""
GPT-OSS Content Classification Module for Censor-AI

This module provides content classification using either OpenAI API or 
Hugging Face Transformers (offline/open model) as backends.
"""

import json
import os
import sys
from typing import Dict, List, Optional, Union
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Try to import optional dependencies
try:
    import openai
    OPENAI_AVAILABLE = True
except ImportError:
    OPENAI_AVAILABLE = False
    logger.warning("OpenAI library not available. Install with: pip install openai")

try:
    from transformers import AutoModelForCausalLM, AutoTokenizer
    import torch
    TRANSFORMERS_AVAILABLE = True
except ImportError:
    TRANSFORMERS_AVAILABLE = False
    logger.warning("Transformers library not available. Install with: pip install transformers torch")


class GPTOSSClassifier:
    """GPT-OSS Content Classifier with multiple backend support."""
    
    def __init__(self, backend: str = "auto", model_name: Optional[str] = None):
        """
        Initialize the classifier with specified backend.
        
        Args:
            backend: "openai", "huggingface", or "auto"
            model_name: Model name for Hugging Face backend
        """
        self.backend = backend
        self.model_name = model_name or "microsoft/DialoGPT-large"  # Fallback model
        self.model = None
        self.tokenizer = None
        self.openai_client = None
        
        # Initialize based on backend preference
        if backend == "auto":
            self._auto_select_backend()
        elif backend == "openai":
            self._init_openai()
        elif backend == "huggingface":
            self._init_huggingface()
        else:
            raise ValueError(f"Unsupported backend: {backend}")
    
    def _auto_select_backend(self):
        """Auto-select the best available backend."""
        openai_key = os.getenv("OPENAI_API_KEY")
        
        if OPENAI_AVAILABLE and openai_key:
            logger.info("Auto-selecting OpenAI backend")
            self.backend = "openai"
            self._init_openai()
        elif TRANSFORMERS_AVAILABLE:
            logger.info("Auto-selecting Hugging Face backend")
            self.backend = "huggingface"
            self._init_huggingface()
        else:
            raise RuntimeError("No suitable backend available. Install openai or transformers library.")
    
    def _init_openai(self):
        """Initialize OpenAI backend."""
        if not OPENAI_AVAILABLE:
            raise RuntimeError("OpenAI library not available")
        
        api_key = os.getenv("OPENAI_API_KEY")
        if not api_key:
            raise RuntimeError("OPENAI_API_KEY environment variable not set")
        
        self.openai_client = openai.OpenAI(api_key=api_key)
        logger.info("OpenAI backend initialized")
    
    def _init_huggingface(self):
        """Initialize Hugging Face backend."""
        if not TRANSFORMERS_AVAILABLE:
            raise RuntimeError("Transformers library not available")
        
        try:
            logger.info(f"Loading model: {self.model_name}")
            self.tokenizer = AutoTokenizer.from_pretrained(self.model_name)
            
            # Configure model loading based on available resources
            device_map = "auto" if torch.cuda.is_available() else "cpu"
            torch_dtype = torch.float16 if torch.cuda.is_available() else torch.float32
            
            self.model = AutoModelForCausalLM.from_pretrained(
                self.model_name,
                device_map=device_map,
                torch_dtype=torch_dtype,
                trust_remote_code=True
            )
            logger.info("Hugging Face backend initialized")
            
        except Exception as e:
            logger.error(f"Failed to load model {self.model_name}: {e}")
            # Fallback to a smaller, more reliable model
            fallback_model = "gpt2"
            logger.info(f"Falling back to {fallback_model}")
            
            self.tokenizer = AutoTokenizer.from_pretrained(fallback_model)
            self.model = AutoModelForCausalLM.from_pretrained(fallback_model)
            
            # Add padding token if it doesn't exist
            if self.tokenizer.pad_token is None:
                self.tokenizer.pad_token = self.tokenizer.eos_token
    
    def _create_classification_prompt(self, metadata: Dict, transcript: str, vision_labels: List[str]) -> str:
        """Create a structured prompt for content classification."""
        
        prompt = """You are a content rating classifier. Analyze the provided content and assign an appropriate age rating.

RATING GUIDELINES:
- 6+: Minimal, non-detailed violence. No nudity. Family-friendly content.
- 12+: Moderate violence without injury detail. Brief, non-sexual nudity. Mild language.
- 16+: Intense but non-gratuitous violence. Partial nudity and implied sexual content. Strong language.
- 18+: Explicit violence with gore. Nudity, including sexual content. Very strong language.

CONTENT TO ANALYZE:
"""
        
        # Add metadata
        if metadata:
            prompt += f"\nMETADATA: {json.dumps(metadata, indent=2)}"
        
        # Add transcript
        if transcript:
            prompt += f"\nTRANSCRIPT: {transcript[:1000]}..."  # Limit transcript length
        
        # Add vision labels
        if vision_labels:
            prompt += f"\nVISION LABELS: {', '.join(vision_labels)}"
        
        prompt += """\n\nPlease respond with ONLY a valid JSON object in this exact format:
{
  "rating": "6+ / 12+ / 16+ / 18+",
  "reason": "short explanation of why this rating was assigned"
}"""
        
        return prompt
    
    def _query_openai(self, prompt: str) -> str:
        """Query OpenAI API."""
        try:
            response = self.openai_client.chat.completions.create(
                model="gpt-4o-mini",  # Use the more cost-effective model
                messages=[
                    {"role": "system", "content": "You are a content rating classifier. Always respond with valid JSON."},
                    {"role": "user", "content": prompt}
                ],
                max_tokens=150,
                temperature=0.1
            )
            return response.choices[0].message.content
        except Exception as e:
            logger.error(f"OpenAI API error: {e}")
            raise
    
    def _query_huggingface(self, prompt: str) -> str:
        """Query Hugging Face model."""
        try:
            # Prepare input
            inputs = self.tokenizer(prompt, return_tensors="pt", truncation=True, max_length=1024)
            
            # Move to appropriate device
            if torch.cuda.is_available() and hasattr(self.model, 'device'):
                inputs = {k: v.to(self.model.device) for k, v in inputs.items()}
            
            # Generate response
            with torch.no_grad():
                outputs = self.model.generate(
                    **inputs,
                    max_new_tokens=150,
                    temperature=0.1,
                    do_sample=True,
                    pad_token_id=self.tokenizer.eos_token_id
                )
            
            # Decode response
            response = self.tokenizer.decode(outputs[0], skip_special_tokens=True)
            
            # Extract only the generated part (after the prompt)
            generated_text = response[len(prompt):].strip()
            return generated_text
            
        except Exception as e:
            logger.error(f"Hugging Face model error: {e}")
            # Return a fallback response
            return '{"rating": "12+", "reason": "Unable to classify content due to model error"}'
    
    def _parse_response(self, response: str) -> Dict[str, str]:
        """Parse the model response and extract rating information."""
        try:
            # Try to find JSON in the response
            start_idx = response.find('{')
            end_idx = response.rfind('}') + 1
            
            if start_idx != -1 and end_idx != 0:
                json_str = response[start_idx:end_idx]
                result = json.loads(json_str)
                
                # Validate required fields
                if "rating" not in result or "reason" not in result:
                    raise ValueError("Missing required fields in response")
                
                # Validate rating format
                valid_ratings = ["6+", "12+", "16+", "18+"]
                if result["rating"] not in valid_ratings:
                    logger.warning(f"Invalid rating '{result['rating']}', defaulting to 12+")
                    result["rating"] = "12+"
                
                return result
            else:
                raise ValueError("No JSON found in response")
                
        except (json.JSONDecodeError, ValueError) as e:
            logger.error(f"Failed to parse response: {e}")
            logger.error(f"Raw response: {response}")
            
            # Return a safe default
            return {
                "rating": "12+",
                "reason": "Unable to parse classification response"
            }
    
    def classify_content(self, metadata: Dict, transcript: str, vision_labels: List[str]) -> Dict[str, str]:
        """
        Classify content and return rating information.
        
        Args:
            metadata: Video metadata dictionary
            transcript: Audio transcript text
            vision_labels: List of vision analysis labels
            
        Returns:
            Dictionary with 'rating' and 'reason' keys
        """
        try:
            # Create classification prompt
            prompt = self._create_classification_prompt(metadata, transcript, vision_labels)
            
            # Query appropriate backend
            if self.backend == "openai":
                response = self._query_openai(prompt)
            elif self.backend == "huggingface":
                response = self._query_huggingface(prompt)
            else:
                raise ValueError(f"Unknown backend: {self.backend}")
            
            # Parse and return result
            result = self._parse_response(response)
            logger.info(f"Classification result: {result}")
            
            return result
            
        except Exception as e:
            logger.error(f"Classification error: {e}")
            return {
                "rating": "18+",  # Conservative fallback
                "reason": f"Classification failed: {str(e)}"
            }


def main():
    """CLI interface for testing the classifier."""
    if len(sys.argv) < 2:
        print("Usage: python oss_classifier.py <json_input>")
        print("Example JSON input: {'metadata': {}, 'transcript': 'text', 'vision_labels': ['label1']}")
        sys.exit(1)
    
    try:
        # Parse input JSON
        input_data = json.loads(sys.argv[1])
        
        metadata = input_data.get("metadata", {})
        transcript = input_data.get("transcript", "")
        vision_labels = input_data.get("vision_labels", [])
        
        # Initialize classifier
        classifier = GPTOSSClassifier(backend="auto")
        
        # Classify content
        result = classifier.classify_content(metadata, transcript, vision_labels)
        
        # Output result
        print(json.dumps(result, indent=2))
        
    except json.JSONDecodeError:
        print("Error: Invalid JSON input")
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
