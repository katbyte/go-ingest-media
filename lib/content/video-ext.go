package content

import (
	"path/filepath"
)

// all known video extensions
var videoExtensions = [...]string{
	".mkv",  // Matroska VideoFile, a flexible open standard format
	".mp4",  // MPEG-4 Part 14, widely used for digital video
	".avi",  // AudioStreams VideoFile Interleave, introduced by Microsoft
	".mpeg", // MPEG-1 and MPEG-2 video, an older standard
	".mov",  // Apple QuickTime Movie, native to Apple devices
	".m4v",  // M4V, Apple’s video file format often used for TV shows, movies, and music videos from iTunes
	".wmv",  // Windows Media VideoFile, Microsoft's video encoding solution
	".webm", // WebM, optimized for the web, supported by HTML5
	".flv",  // Flash VideoFile, used by Adobe Flash Player
	".rmvb", // RealMedia Variable Bitrate, by RealNetworks
	".3gp",  // 3GPP, used for video on mobile phones
}

// map of extension to index in videoExtensions
var videoExtensionsMap = map[string]int{}

func init() {
	for i, ext := range videoExtensions {
		videoExtensionsMap[ext] = i
	}
}

func IsVideoFile(path string) bool {
	for _, ext := range videoExtensions {
		if filepath.Ext(path) == ext {
			return true
		}
	}

	return false
}

func VideoExtensionIndex(path string) int {
	ext := filepath.Ext(path)
	if i, ok := videoExtensionsMap[ext]; ok {
		return i
	}

	return -1
}
