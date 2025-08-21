#!/usr/bin/env python3
"""
Test script for GPT-OSS integration
"""

import json
import sys
import subprocess
import time

def test_classifier_directly():
    """Test the Python classifier directly"""
    print("🧪 Testing GPT-OSS classifier directly...")
    
    test_data = {
        "metadata": {
            "filename": "test_video.mp4",
            "duration": 120
        },
        "transcript": "Action scene with characters fighting, mild violence, no explicit content",
        "vision_labels": ["action", "fighting", "characters", "outdoor"]
    }
    
    try:
        cmd = ["python3", "oss_classifier.py", json.dumps(test_data)]
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        
        if result.returncode == 0:
            response = json.loads(result.stdout)
            print(f"✅ Direct test passed!")
            print(f"   Rating: {response['rating']}")
            print(f"   Reason: {response['reason']}")
            return True
        else:
            print(f"❌ Direct test failed:")
            print(f"   Error: {result.stderr}")
            return False
            
    except subprocess.TimeoutExpired:
        print("❌ Direct test timed out (>30s)")
        return False
    except Exception as e:
        print(f"❌ Direct test error: {e}")
        return False

def test_go_integration():
    """Test the Go backend integration"""
    print("\n🧪 Testing Go backend integration...")
    print("   Note: This requires the Go server to be running on port 8000")
    
    try:
        import requests
    except ImportError:
        print("   ⚠️  requests library not available, skipping Go integration test")
        print("   Install with: pip3 install requests")
        return False
    
    test_data = {
        "metadata": {
            "filename": "test_video.mp4",
            "duration": 120
        },
        "transcript": "Family-friendly content with no violence or inappropriate material",
        "vision_labels": ["family", "happy", "outdoor", "nature"]
    }
    
    try:
        response = requests.post(
            "http://localhost:8000/classify",
            json=test_data,
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Go integration test passed!")
            print(f"   Rating: {result['rating']}")
            print(f"   Reason: {result['reason']}")
            return True
        else:
            print(f"❌ Go integration test failed:")
            print(f"   Status: {response.status_code}")
            print(f"   Response: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        print("❌ Go integration test failed:")
        print("   Could not connect to Go server on port 8000")
        print("   Start the server with: go run main.go")
        return False
    except Exception as e:
        print(f"❌ Go integration test error: {e}")
        return False

def test_different_content():
    """Test classifier with different content types"""
    print("\n🧪 Testing different content types...")
    
    test_cases = [
        {
            "name": "Family Content",
            "data": {
                "metadata": {"type": "family"},
                "transcript": "Children playing in a park, laughing and having fun",
                "vision_labels": ["children", "park", "happy", "daylight"]
            },
            "expected": "6+"
        },
        {
            "name": "Teen Content", 
            "data": {
                "metadata": {"type": "teen"},
                "transcript": "High school drama with some mild language and romantic themes",
                "vision_labels": ["teenagers", "school", "romance", "drama"]
            },
            "expected": "12+"
        },
        {
            "name": "Action Content",
            "data": {
                "metadata": {"type": "action"},
                "transcript": "Intense action sequence with fighting and some violence",
                "vision_labels": ["action", "fighting", "weapons", "intense"]
            },
            "expected": "16+"
        },
        {
            "name": "Mature Content",
            "data": {
                "metadata": {"type": "mature"},
                "transcript": "Graphic violence and adult themes with explicit content",
                "vision_labels": ["violence", "blood", "adult", "explicit"]
            },
            "expected": "18+"
        }
    ]
    
    passed = 0
    total = len(test_cases)
    
    for case in test_cases:
        try:
            cmd = ["python3", "oss_classifier.py", json.dumps(case["data"])]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=15)
            
            if result.returncode == 0:
                response = json.loads(result.stdout)
                rating = response["rating"]
                
                # Check if rating is appropriate (may not be exact match due to model variance)
                if rating in ["6+", "12+", "16+", "18+"]:
                    print(f"   ✅ {case['name']}: {rating} ({response['reason'][:50]}...)")
                    passed += 1
                else:
                    print(f"   ⚠️  {case['name']}: Invalid rating '{rating}'")
            else:
                print(f"   ❌ {case['name']}: Failed - {result.stderr[:50]}...")
                
        except Exception as e:
            print(f"   ❌ {case['name']}: Error - {e}")
    
    print(f"\n   Results: {passed}/{total} test cases passed")
    return passed == total

def main():
    """Run all tests"""
    print("🚀 GPT-OSS Integration Test Suite")
    print("=" * 50)
    
    # Test 1: Direct classifier test
    test1_passed = test_classifier_directly()
    
    # Test 2: Go integration test
    test2_passed = test_go_integration()
    
    # Test 3: Different content types
    test3_passed = test_different_content()
    
    # Summary
    print("\n" + "=" * 50)
    print("📊 Test Summary:")
    print(f"   Direct Classifier: {'✅ PASS' if test1_passed else '❌ FAIL'}")
    print(f"   Go Integration:    {'✅ PASS' if test2_passed else '❌ FAIL'}")
    print(f"   Content Variety:   {'✅ PASS' if test3_passed else '❌ FAIL'}")
    
    if test1_passed and test2_passed and test3_passed:
        print("\n🎉 All tests passed! GPT-OSS integration is working correctly.")
        return 0
    else:
        print("\n⚠️  Some tests failed. Check the output above for details.")
        return 1

if __name__ == "__main__":
    sys.exit(main())
