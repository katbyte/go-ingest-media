package content

import (
	"fmt"
	"os"
	"path/filepath"
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
}

// lazy "close enough compare"
func (v VideoFile) IsBasicallyTheSameTo(v2 VideoFile) bool {
	return v.Ext == v2.Ext &&
		v.SizeBytes == v2.SizeBytes &&
		v.Duration == v2.Duration &&
		v.BitRate == v2.BitRate &&
		v.Resolution == v2.Resolution &&
		v.VideoStream.CodecName == v2.VideoStream.CodecName &&
		v.VideoStream.Profile == v2.VideoStream.Profile
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
		return nil, fmt.Errorf("error getting video info: %w", err)
	}

	v.BitRate, err = strconv.Atoi(probe.Format.BitRate)
	if err != nil {
		return nil, err
	}

	v.Duration, err = strconv.ParseFloat(probe.Format.Duration, 64)
	if err != nil {
		return nil, err
	}

	vStreams, err := probe.VideoStreams()
	if err != nil {
		return nil, fmt.Errorf("error getting video streams: %w", err)
	}
	if len(vStreams) != 1 {
		return nil, fmt.Errorf("expected 1 video stream, found %d", len(vStreams))
	}
	v.VideoStream = vStreams[0]
	v.Resolution = fmt.Sprintf("%dx%d", v.VideoStream.Width, v.VideoStream.Height)
	v.ResolutionW = v.VideoStream.Width
	v.ResolutionH = v.VideoStream.Height

	v.AudioStreams, err = probe.AudioStreams()
	if err != nil {
		return nil, fmt.Errorf("error getting audio streams: %w", err)
	}
	if len(v.AudioStreams) < 1 {
		return nil, fmt.Errorf("expected at least 1 audio stream, found %d", len(v.AudioStreams))
	}

	return &v, nil
}
