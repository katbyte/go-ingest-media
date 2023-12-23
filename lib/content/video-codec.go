package content

import (
	"path/filepath"
)

// list video codecs in order of preference
var videoCodecs = [...]string{
	"hevc",       // H.265 or HEVC (High Efficiency Video Coding)
	"h264",       // H.264 or AVC (Advanced Video Coding)
	"av1",        // AV1, AOMedia Video 1, designed for internet streaming
	"mpeg4",      // MPEG-4 Part 2, used for video compression
	"vp8",        // VP8, an open video codec developed by Google
	"vp9",        // VP9, successor to VP8 with better compression
	"mpeg2video", // MPEG-2, used in DVDs and broadcast TV
	"theora",     // Theora, an open video codec by the Xiph.Org Foundation
	"wmv",        // Windows Media Video, developed by Microsoft
	"h263",       // H.263, an older codec used in video conferencing
}

// map of extension to index in videoExtensions
var videoCodecMap = map[string]int{}

func init() {
	for i, codec := range videoCodecs {
		videoCodecMap[codec] = i
	}
}

func VideoCodecIndex(path string) int {
	ext := filepath.Ext(path)
	if i, ok := videoExtensionsMap[ext]; ok {
		return i
	}

	return -1
}
