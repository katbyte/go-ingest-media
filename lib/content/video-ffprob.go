package content

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strconv"
)

// FFProbeOutput represents the structure of the output from ffprobe
type FFProbeOutput struct {
	Format  FFProbeFormat   `json:"format"`
	Streams []FFProbeStream `json:"streams"`
}

// FFProbeFormat contains general information about the media
type FFProbeFormat struct {
	Filename       string            `json:"filename"`
	NumStreams     int               `json:"nb_streams"`
	NumPrograms    int               `json:"nb_programs"`
	FormatName     string            `json:"format_name"`
	FormatLongName string            `json:"format_long_name"`
	StartTime      string            `json:"start_time"`
	Duration       string            `json:"duration"`
	Size           string            `json:"size"`
	BitRate        string            `json:"bit_rate"`
	ProbeScore     int               `json:"probe_score"`
	Tags           map[string]string `json:"tags"`
}

// FFProbeStream contains details for each individual stream
type FFProbeStream struct {
	Index              int               `json:"index"`
	CodecName          string            `json:"codec_name"`
	CodecLongName      string            `json:"codec_long_name"`
	Profile            string            `json:"profile"`
	CodecType          string            `json:"codec_type"`
	CodecTimeBase      string            `json:"codec_time_base"`
	CodecTagString     string            `json:"codec_tag_string"`
	CodecTag           string            `json:"codec_tag"`
	Disposition        map[string]int    `json:"disposition"`
	Width              int               `json:"width,omitempty"`
	Height             int               `json:"height,omitempty"`
	DisplayAspectRatio string            `json:"display_aspect_ratio,omitempty"`
	PixFmt             string            `json:"pix_fmt,omitempty"`
	Level              int               `json:"level,omitempty"`
	ColorRange         string            `json:"color_range,omitempty"`
	ColorSpace         string            `json:"color_space,omitempty"`
	ColorTransfer      string            `json:"color_transfer,omitempty"`
	ColorPrimaries     string            `json:"color_primaries,omitempty"`
	ChromaLocation     string            `json:"chroma_location,omitempty"`
	FieldOrder         string            `json:"field_order,omitempty"`
	TimeBase           string            `json:"time_base"`
	StartPts           int64             `json:"start_pts"`
	StartTime          string            `json:"start_time"`
	DurationTs         int64             `json:"duration_ts"`
	Duration           string            `json:"duration"`
	BitRate            string            `json:"bit_rate,omitempty"`
	MaxBitRate         string            `json:"max_bit_rate,omitempty"`
	BitsPerRawSample   string            `json:"bits_per_raw_sample,omitempty"`
	NbFrames           string            `json:"nb_frames"`
	NbReadFrames       string            `json:"nb_read_frames,omitempty"`
	NbReadPackets      string            `json:"nb_read_packets,omitempty"`
	Tags               map[string]string `json:"tags"`
	RFrameRate         string            `json:"r_frame_rate,omitempty"`
	SampleFmt          string            `json:"sample_fmt,omitempty"`
	SampleRate         string            `json:"sample_rate,omitempty"`
	Channels           int               `json:"channels,omitempty"`
	ChannelLayout      string            `json:"channel_layout,omitempty"`
	BitsPerSample      int               `json:"bits_per_sample,omitempty"`
	// Add other fields as needed
}

// GetVideoInfo runs ffprobe on the specified video file and returns its information.
func FFProbe(pathToVideo string) (*FFProbeOutput, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", pathToVideo)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	var info FFProbeOutput
	err = json.Unmarshal(out.Bytes(), &info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

type FFProbeStreamVideo struct {
	Index              int               `json:"index"`
	CodecName          string            `json:"codec_name"`
	CodecLongName      string            `json:"codec_long_name"`
	Profile            string            `json:"profile"`
	Width              int               `json:"width"`
	Height             int               `json:"height"`
	DisplayAspectRatio string            `json:"display_aspect_ratio"`
	PixFmt             string            `json:"pix_fmt"`
	FrameRate          string            `json:"r_frame_rate"`
	Duration           float64           `json:"duration"`
	BitRate            int               `json:"bit_rate"`
	Tags               map[string]string `json:"tags"`
}

func (output *FFProbeOutput) VideoStreams() ([]FFProbeStreamVideo, error) {
	var videoStreams []FFProbeStreamVideo
	for _, s := range output.Streams {
		if s.CodecType == "video" && s.Disposition["attached_pic"] != 1 {
			vs := FFProbeStreamVideo{
				Index:              s.Index,
				CodecName:          s.CodecName,
				CodecLongName:      s.CodecLongName,
				Width:              s.Width,
				Height:             s.Height,
				DisplayAspectRatio: s.DisplayAspectRatio,
				PixFmt:             s.PixFmt,
				FrameRate:          s.RFrameRate,
				Profile:            s.Profile,
				Tags:               s.Tags,
			}

			var err error
			if s.BitRate != "" {
				vs.BitRate, err = strconv.Atoi(s.BitRate)
				if err != nil {
					return nil, err
				}
			}
			if s.Duration != "" {
				vs.Duration, err = strconv.ParseFloat(s.Duration, 64)
				if err != nil {
					return nil, err
				}
			}
			videoStreams = append(videoStreams, vs)
		}
	}
	return videoStreams, nil
}

type FFProbeStreamAudio struct {
	Index         int               `json:"index"`
	CodecName     string            `json:"codec_name"`
	CodecLongName string            `json:"codec_long_name"`
	SampleRate    string            `json:"sample_rate"`
	Channels      int               `json:"channels"`
	ChannelLayout string            `json:"channel_layout"`
	Duration      float64           `json:"duration"`
	BitRate       int               `json:"bit_rate"`
	Tags          map[string]string `json:"tags"`
	Language      string            `json:"language"`
}

func (output *FFProbeOutput) AudioStreams() ([]FFProbeStreamAudio, error) {
	var audioStreams []FFProbeStreamAudio
	for _, s := range output.Streams {
		if s.CodecType == "audio" {
			as := FFProbeStreamAudio{
				Index:         s.Index,
				CodecName:     s.CodecName,
				CodecLongName: s.CodecLongName,
				SampleRate:    s.SampleRate,
				Channels:      s.Channels,
				ChannelLayout: s.ChannelLayout,
				Tags:          s.Tags,
			}

			var err error
			if s.BitRate != "" {
				as.BitRate, err = strconv.Atoi(s.BitRate)
				if err != nil {
					return nil, err
				}
			}

			if s.Duration != "" {
				as.Duration, err = strconv.ParseFloat(s.Duration, 64)
				if err != nil {
					return nil, err
				}
			}

			if language, ok := s.Tags["language"]; ok {
				as.Language = language
			}
			audioStreams = append(audioStreams, as)
		}
	}

	return audioStreams, nil
}

type FFProbeStreamImage struct {
	Index         int               `json:"index"`
	CodecName     string            `json:"codec_name"`
	CodecLongName string            `json:"codec_long_name"`
	Width         int               `json:"width,omitempty"`
	Height        int               `json:"height,omitempty"`
	Tags          map[string]string `json:"tags"`
}

func (output *FFProbeOutput) ImageStreams() ([]FFProbeStreamImage, error) {
	var imageStreams []FFProbeStreamImage
	for _, s := range output.Streams {
		isImageStream := s.CodecType == "image"
		isAttachedPic := s.CodecType == "video" && s.Disposition["attached_pic"] == 1

		if isImageStream || isAttachedPic {
			imgStream := FFProbeStreamImage{
				Index:         s.Index,
				CodecName:     s.CodecName,
				CodecLongName: s.CodecLongName,
				Width:         s.Width,
				Height:        s.Height,
				Tags:          s.Tags,
			}
			imageStreams = append(imageStreams, imgStream)
		}
	}
	return imageStreams, nil
}

type FFProbeStreamSubtitle struct {
	Index         int               `json:"index"`
	CodecName     string            `json:"codec_name"`
	CodecLongName string            `json:"codec_long_name"`
	Language      string            `json:"language,omitempty"`
	Tags          map[string]string `json:"tags"`
}

func (output *FFProbeOutput) SubtitleStreams() ([]FFProbeStreamSubtitle, error) {
	var subtitleStreams []FFProbeStreamSubtitle
	for _, s := range output.Streams {
		if s.CodecType == "subtitle" { // Adjust this condition based on how FFProbe marks subtitle streams
			subStream := FFProbeStreamSubtitle{
				Index:         s.Index,
				CodecName:     s.CodecName,
				CodecLongName: s.CodecLongName,
				Tags:          s.Tags,
			}
			if language, ok := s.Tags["language"]; ok {
				subStream.Language = language
			}
			subtitleStreams = append(subtitleStreams, subStream)
		}
	}
	return subtitleStreams, nil
}
