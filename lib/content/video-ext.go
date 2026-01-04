package content

import (
	"path/filepath"
)

// all known video extensions
var videoExtensions = [...]string{
	".mkv",  // Matroska VideoFile, a flexible open standard format
	".mp4",  // MPEG-4 Part 14, widely used for digital video
	".avi",  // AudioStreams VideoFile Interleave, introduced by Microsoft
	".mov",  // Apple QuickTime Movie, native to Apple devices
	".mpeg", // MPEG-1 and MPEG-2 video, an older standard
	".mpg",  // MPEG-1 and MPEG-2 video, an older standard
	".mov",  // Apple QuickTime Movie, native to Apple devices
	".m4v",  // M4V, Apple’s video file format often used for TV shows, movies, and music videos from iTunes
	".wmv",  // Windows Media VideoFile, Microsoft's video encoding solution
	".webm", // WebM, optimised for the web, supported by HTML5
	".flv",  // Flash VideoFile, used by Adobe Flash Player
	".rmvb", // RealMedia Variable Bitrate, by RealNetworks
	".3gp",  // 3GPP, used for video on mobile phones
	".3g2",  // 3GPP2, used for video on mobile phones
	".vob",  // VOB, used for DVDs
	".ts",   // MPEG Transport Stream, used for streaming video data
	".m2ts", // Blu-ray BDAV MPEG-2 Transport Stream
	".mts",  // AVCHD MPEG-2 Transport Stream
	".mxf",  // Material eXchange Format, used by professional video cameras
	".ogv",  // Ogg VideoFile, used for HTML5 video
	".ogm",  // Ogg Media, used for HTML5 video
	".rm",   // RealMedia, used for streaming video
	".divx", // DivX, a video codec
	".xvid", // Xvid, a video codec
	".asf",  // Advanced Systems Format, Microsoft's proprietary digital audio/digital video container format
	".drc",  // Dirac, a video codec
	".f4v",  // Flash VideoFile, used by Adobe Flash Player
	".f4p",  // Flash VideoFile, used by Adobe Flash Player
	".f4a",  // Flash VideoFile, used by Adobe Flash Player
	".f4b",  // Flash VideoFile, used by Adobe Flash Player
	".gifv", // HTML5 video
	".gif",  // HTML5 video
	".mng",  // HTML5 video
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
