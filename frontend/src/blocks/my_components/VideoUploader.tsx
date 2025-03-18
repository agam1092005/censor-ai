import { useState, useEffect } from "react";
import { useDropzone } from "react-dropzone";
import { Card } from "@/components/ui/card";
import { UploadCloud, Loader, Scissors, EyeOff, ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function VideoUploader() {
  const [video, setVideo] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [ratings, setRatings] = useState<any[]>([]);
  const [analyzed, setAnalyzed] = useState(false);
  const [selectedRating, setSelectedRating] = useState<string | null>(null);
  const [selectedOption, setSelectedOption] = useState<string | null>(null);

  // Auto-select trim for 6+ rating
  useEffect(() => {
    if (selectedRating === "6+" && !selectedOption) {
      setSelectedOption('trim');
    }
  }, [selectedRating, selectedOption]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    accept: { "video/*": [] },
    multiple: false,
    onDrop: (acceptedFiles: File[]) => {
      setVideo(acceptedFiles[0]);
      setAnalyzed(false);
      setSelectedRating(null);
      setSelectedOption(null);
    },
    disabled: loading || !!video,
  });

  const handleAnalyze = async () => {
    if (!video) return;
    setLoading(true);
    setRatings([]);
    setSelectedRating(null);
    setSelectedOption(null);

    const formData = new FormData();
    formData.append("video", video);

    try {
      const response = await fetch("http://localhost:8000/upload", {
        method: "POST",
        body: formData,
      });

      const data = await response.json();
      console.log("Response Data:", data);

      if (data.ratings) {
        setRatings(data.ratings);
        setAnalyzed(true);
      }
    } catch (error) {
      console.error("Error:", error);
    } finally {
      setLoading(false);
    }
  };

  const resetUpload = () => {
    setVideo(null);
    setRatings([]);
    setAnalyzed(false);
    setSelectedRating(null);
    setSelectedOption(null);
  };

  const handleConvert = () => {
    if (!selectedRating || !selectedOption) return;
    
    // Here you would implement the actual conversion logic
    console.log(`Converting to ${selectedRating} using ${selectedOption} option`);
    
    // For now, just show an alert
    alert(`Video will be converted to ${selectedRating} using ${selectedOption} option`);
  };

  // Function to get the highest age rating
  const getHighestRating = () => {
    if (!ratings.length) return null;
    
    const ratingValues: { [key: string]: number } = {
      "6+": 6,
      "12+": 12,
      "16+": 16,
      "18+": 18
    };
    
    let highestRating = "6+";
    
    ratings.forEach(rating => {
      if (ratingValues[rating.rating] > ratingValues[highestRating]) {
        highestRating = rating.rating;
      }
    });
    
    return highestRating;
  };

  // Function to get top 3 most common content warnings for the highest rating
  const getTopContentWarnings = () => {
    if (!ratings.length) return [];
    
    const highestRating = getHighestRating();
    const highestRatedSegments = ratings.filter(r => r.rating === highestRating);
    
    // Extract all content warnings and count occurrences
    const allWarnings: string[] = [];
    highestRatedSegments.forEach(segment => {
      const notes = segment.notes.split(',').map((note: string) => note.trim());
      allWarnings.push(...notes);
    });
    
    // Count occurrences
    const warningCounts: { [key: string]: number } = {};
    allWarnings.forEach(warning => {
      warningCounts[warning] = (warningCounts[warning] || 0) + 1;
    });
    
    // Sort by count and get top 3
    return Object.keys(warningCounts)
      .sort((a, b) => warningCounts[b] - warningCounts[a])
      .slice(0, 3);
  };

  // Get conversion options based on highest rating
  const getConversionOptions = () => {
    const highestRating = getHighestRating();
    
    switch(highestRating) {
      case "18+":
        return ["6+", "12+", "16+"];
      case "16+":
        return ["6+", "12+"];
      case "12+":
        return ["6+"];
      default:
        return [];
    }
  };

  // Determine which content to show in the glassy box
  const renderContent = () => {
    if (loading) {
      return (
        <div className="flex flex-col items-center justify-center h-full py-4 px-4">
          <Loader className="animate-spin text-white mb-4" size={40} />
          <p className="text-white text-center">
            Analysis in progress. This will take a few minutes...
          </p>
        </div>
      );
    }

    if (analyzed && ratings.length > 0) {
      return (
        <div className="h-full w-full flex flex-col items-center">
          {getHighestRating() === "6+" ? (
            <p className="text-lg text-white mb-2 text-center">
              This video is suitable for all ages. No scenes of violence, blood, nudity found.
            </p>
          ) : (
            <p className="text-lg text-white mb-2 text-center">
              This video is minimum rated for <span className="font-bold">{getHighestRating()}</span> age due to found scenes of {getTopContentWarnings().join(', ')}, etc.
            </p>
          )}

          {getConversionOptions().length > 0 && (
            <div className="w-full flex flex-col items-center">
              <p className="text-lg mb-2 text-white text-center">Convert this video for age-appropriate viewing:</p>
              
              {/* Rating Selection */}
              <div className="flex gap-2 mb-3 justify-center">
                {getConversionOptions().map(option => (
                  console.log(option, selectedOption),
                  <button 
                    key={option} 
                    className={`py-2 px-8 rounded-lg font-medium transition-colors border ${   
                      selectedRating === option 
                        ? 'bg-white text-black border-white' 
                        : 'bg-black/50 text-white border-gray-500'
                    }`}
                    onClick={() => setSelectedRating(option)}
                  >
                    {option}
                  </button>
                ))}
              </div>
              
              {/* Processing Options */}
              {selectedRating && (
                <div className="mb-3">
                  {selectedRating === "6+" ? (
                    <p className="text-white mb-2 text-center">For 6+ rating, only trimming is available to remove sensitive content.</p>
                  ) : (
                    <div className="flex gap-2 justify-center">
                      <button 
                        key={'blur'}
                        className={`flex items-center gap-2 py-2 px-8 rounded-lg font-medium transition-colors border ${
                          selectedOption === 'blur' 
                            ? 'bg-white text-black border-white' 
                            : 'bg-black/50 text-white border-gray-500'
                        }`}
                        onClick={() => setSelectedOption('blur')}
                      >
                        <EyeOff size={16} />
                        <span>Blur</span>
                      </button>
                      <button 
                        key={'trim'}
                        className={`flex items-center gap-2 py-2 px-8 rounded-lg font-medium transition-colors border ${
                          selectedOption === 'trim' 
                            ? 'bg-white text-black border-white' 
                            : 'bg-black/50 text-white border-gray-500'
                        }`}
                        onClick={() => setSelectedOption('trim')}
                      >
                        <Scissors size={16} />
                        <span>Trim</span>
                      </button>
                    </div>
                  )}
                </div>
              )}
              
              {/* Convert Button */}
              {selectedRating && selectedOption && (
                <div className="flex justify-center">
                  <Button 
                    onClick={handleConvert}
                    className="bg-green-600 hover:bg-green-700 text-white border border-green-500 py-2 px-6"
                  >
                    Convert Video
                  </Button>
                </div>
              )}
            </div>
          )}
        </div>
      );
    }

    // Default: Upload state
    return (
      <div 
        {...getRootProps()}
        className={`h-full flex flex-col justify-center items-center py-4 px-4 ${
          loading ? "cursor-not-allowed" : "cursor-pointer"
        }`}
      >
        <input {...getInputProps()} disabled={loading} />
        <div className="flex flex-col items-center">
          <UploadCloud size={50} className="text-white mb-4" />
          <p className="text-white mt-2 text-lg text-center">
            {isDragActive ? "Drop the video here..." : "Drag & drop a video file here, or click to browse"}
          </p>
          {video && <p className="mt-2 text-white text-base">{video.name}</p>}
        </div>
        {video && !analyzed && (
          <Button 
            onClick={(e) => {
              e.stopPropagation();
              handleAnalyze();
            }} 
            className="mt-4 bg-transparent hover:bg-gray-700 text-white border border-white py-2 px-6"
            disabled={loading}
          >
            {loading ? "Analyzing..." : "Analyze"}
          </Button>
        )}
      </div>
    );
  };

  return (
    <div className="w-full max-w-3xl mx-auto flex items-center justify-center">
      <div className="w-full">
        {analyzed && ratings.length > 0 ? (
          <Card
            className="py-4 px-6 border-2 border-white/20 rounded-3xl h-[380px] transition-all duration-300"
            style={{
              background: "rgba(40, 40, 40, 0.7)",
              backdropFilter: "blur(10px)",
              WebkitBackdropFilter: "blur(10px)",
            }}
          >
            {renderContent()}
          </Card>
        ) : (
          <Card
            className="py-6 px-6 border-2 border-white/20 rounded-3xl h-[380px] transition-all duration-300"
            style={{
              background: "rgba(40, 40, 40, 0.7)",
              backdropFilter: "blur(10px)",
              WebkitBackdropFilter: "blur(10px)",
            }}
          >
            {renderContent()}
          </Card>
        )}
      </div>
    </div>
  );
}
