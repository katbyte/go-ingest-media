package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/katbyte/go-ingest-media/lib/ktio"
)

type VideoFile struct {
	// video details
	Path string
	Ext  string

	SizeBytes   int64
	SizeGb      float64
	Duration    float64
	BitRate     int
	Resolution  string
	ResolutionW int
	ResolutionH int

	VideoStream  FFProbeStreamVideo
	AudioStreams []FFProbeStreamAudio
	ImageStreams []FFProbeStreamImage
	Subtitles    []FFProbeStreamSubtitle

	// Set to true if ffprobe failed - only basic file info available
	FFProbeFailed bool
}

// lazy "close enough compare"
func (v VideoFile) IsBasicallyTheSameTo(v2 VideoFile) bool {
	return v.Ext == v2.Ext &&
		v.SizeBytes == v2.SizeBytes &&
		v.Duration == v2.Duration &&
		v.BitRate == v2.BitRate &&
		v.Resolution == v2.Resolution &&
		v.VideoStream.CodecName == v2.VideoStream.CodecName &&
		v.VideoStream.Profile == v2.VideoStream.Profile &&
		len(v.AudioStreams) == len(v2.AudioStreams) &&
		len(v.ImageStreams) == len(v2.ImageStreams) &&
		len(v.Subtitles) == len(v2.Subtitles)
}

func VideosInPath(path string) ([]VideoFile, error) {
	files, err := ktio.ListFiles(path)
	if err != nil {
		return nil, fmt.Errorf("error listing content folders: %w", err)
	}

	videos := make([]VideoFile, 0)
	for _, f := range files {
		if IsVideoFile(f) {
			v, err := VideoFor(f)
			if err != nil {
				return nil, err
			}

			videos = append(videos, *v)
		}
	}

	return videos, nil
}

func VideoFor(path string) (*VideoFile, error) {
	v := VideoFile{
		Path: path,
		Ext:  filepath.Ext(path),
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	v.SizeBytes = fileInfo.Size()
	v.SizeGb = float64(v.SizeBytes) / 1024 / 1024 / 1024

	probe, err := FFProbe(path)
	if err != nil {
		// FFProbe failed - return partial video info with what we have
		v.FFProbeFailed = true
		v.Resolution = "UNKNOWN"
		return &v, nil //nolint:nilerr // intentionally returning nil error for graceful degradation
	}

	v.BitRate, _ = strconv.Atoi(probe.Format.BitRate)
	v.Duration, _ = strconv.ParseFloat(probe.Format.Duration, 64)

	vStreams, err := probe.VideoStreams()
	if err != nil {
		v.FFProbeFailed = true
		v.Resolution = "ERROR"
		return &v, nil //nolint:nilerr // intentionally returning nil error for graceful degradation
	}
	if len(vStreams) == 0 {
		v.FFProbeFailed = true
		v.Resolution = "NO VIDEO"
		return &v, nil
	}
	if len(vStreams) > 1 {
		// Multiple video streams - just use the first one
		v.Resolution = fmt.Sprintf("%dx%d (+%d)", vStreams[0].Width, vStreams[0].Height, len(vStreams)-1)
	} else {
		v.Resolution = fmt.Sprintf("%dx%d", vStreams[0].Width, vStreams[0].Height)
	}
	v.VideoStream = vStreams[0]
	v.ResolutionW = v.VideoStream.Width
	v.ResolutionH = v.VideoStream.Height

	v.AudioStreams, _ = probe.AudioStreams()
	v.ImageStreams, _ = probe.ImageStreams()
	v.Subtitles, _ = probe.SubtitleStreams()

	return &v, nil
}

func (v *VideoFile) AudioStreamsSortedByLanguage() []FFProbeStreamAudio {
	sortedAudioStreams := make([]FFProbeStreamAudio, len(v.AudioStreams))
	copy(sortedAudioStreams, v.AudioStreams)

	sort.Slice(sortedAudioStreams, func(i, j int) bool {
		return sortedAudioStreams[i].Language < sortedAudioStreams[j].Language
	})

	return sortedAudioStreams
}

func (v *VideoFile) SubtitlesSortedByLanguage() []FFProbeStreamSubtitle {
	sortedSubtitles := make([]FFProbeStreamSubtitle, len(v.Subtitles))
	copy(sortedSubtitles, v.Subtitles)

	sort.Slice(sortedSubtitles, func(i, j int) bool {
		return sortedSubtitles[i].Language < sortedSubtitles[j].Language
	})

	return sortedSubtitles
}

// AspectRatio returns the aspect ratio as a simplified string like "16:9" or "4:3"
func (v *VideoFile) AspectRatio() string {
	if v.ResolutionW == 0 || v.ResolutionH == 0 {
		return "unknown"
	}

	// Calculate aspect ratio
	ratio := float64(v.ResolutionW) / float64(v.ResolutionH)

	// Common aspect ratios
	switch {
	case ratio >= 2.3 && ratio <= 2.45:
		return "2.39:1" // Cinemascope
	case ratio >= 2.1 && ratio <= 2.2:
		return "2.2:1" // 70mm
	case ratio >= 1.8 && ratio <= 1.95:
		return "1.85:1" // Widescreen
	case ratio >= 1.7 && ratio <= 1.8:
		return "16:9"
	case ratio >= 1.3 && ratio <= 1.4:
		return "4:3"
	case ratio >= 1.5 && ratio <= 1.55:
		return "3:2"
	default:
		return fmt.Sprintf("%.2f:1", ratio)
	}
}

// IsWidescreen returns true if the video is widescreen (16:9 or wider)
func (v *VideoFile) IsWidescreen() bool {
	if v.ResolutionW == 0 || v.ResolutionH == 0 {
		return false
	}
	ratio := float64(v.ResolutionW) / float64(v.ResolutionH)
	return ratio >= 1.6 // 16:9 = 1.777, 4:3 = 1.333
}
