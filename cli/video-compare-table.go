package cli

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	c "github.com/gookit/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	_ "github.com/mattn/go-sqlite3"
)

var tablestyle = table.Style{
	Name:   "VideoStyle",
	Box:    table.StyleBoxBold,
	Color:  table.ColorOptionsDefault,
	Format: table.FormatOptionsDefault,
	HTML:   table.DefaultHTMLOptions,
	Title:  table.TitleOptionsDefault,
	Options: table.Options{
		DrawBorder:      false,
		SeparateColumns: true,
		SeparateFooter:  true,
		SeparateHeader:  true,
		SeparateRows:    false,
	},
}

type TableRow struct {
	Name       string
	Value      func(v content.VideoFile) string
	Equal      func(v1, v2 content.VideoFile) bool
	BetterThan func(v1, v2 content.VideoFile) bool
}

var rows = []TableRow{
	{
		"Ext",
		func(file content.VideoFile) string { return file.Ext },
		func(v1, v2 content.VideoFile) bool { return v1.Ext == v2.Ext },
		func(v1, v2 content.VideoFile) bool {
			return content.VideoExtensionIndex(v1.Ext) > content.VideoExtensionIndex(v2.Ext)
		},
	},
	{
		"Size",
		func(file content.VideoFile) string { return fmt.Sprintf("%0.2f", file.SizeGb) },
		func(v1, v2 content.VideoFile) bool { return v1.SizeGb == v2.SizeGb },
		func(v1, v2 content.VideoFile) bool { return v1.SizeGb < v2.SizeGb },
	},
	{
		"Resolution",
		func(file content.VideoFile) string { return file.Resolution },
		func(v1, v2 content.VideoFile) bool { return v1.Resolution == v2.Resolution },
		func(v1, v2 content.VideoFile) bool {
			return v1.ResolutionW*v1.ResolutionH > v2.ResolutionW*v2.ResolutionH
		},
	},
	{
		"Codec",
		func(file content.VideoFile) string { return file.VideoStream.CodecName },
		func(v1, v2 content.VideoFile) bool { return v1.VideoStream.CodecName == v2.VideoStream.CodecName },
		func(v1, v2 content.VideoFile) bool { return v1.VideoStream.CodecName > v2.VideoStream.CodecName },
	},
	{
		"Profile",
		func(file content.VideoFile) string { return file.VideoStream.Profile },
		func(v1, v2 content.VideoFile) bool { return v1.VideoStream.Profile == v2.VideoStream.Profile },
		func(v1, v2 content.VideoFile) bool { return v1.VideoStream.Profile > v2.VideoStream.Profile },
	},
	{
		"Duration",
		func(file content.VideoFile) string { return fmt.Sprintf("%0.2f", file.Duration) },
		func(v1, v2 content.VideoFile) bool { return v1.Duration == v2.Duration },
		func(v1, v2 content.VideoFile) bool { return v1.Duration > v2.Duration },
	},
	{
		"Bitrate",
		func(file content.VideoFile) string { return strconv.Itoa(file.BitRate) },
		func(v1, v2 content.VideoFile) bool { return v1.BitRate == v2.BitRate },
		func(v1, v2 content.VideoFile) bool { return v1.BitRate > v2.BitRate },
	},
}

func RenderVideoComparisonTable(indent int, headers []string, videos []content.VideoFile) {
	var buf bytes.Buffer
	t := table.NewWriter()
	t.SetOutputMirror(&buf)
	t.SetStyle(tablestyle)

	same := true
	if len(videos) > 1 {
		for _, v := range videos[1:] {
			same = same && videos[0].IsBasicallyTheSameTo(v)
		}
	} else {
		same = false
	}

	headerRow := table.Row{"", headers[0]}
	for _, h := range headers[1:] {
		headerRow = append(headerRow, h)
	}

	t.AppendHeader(headerRow, table.RowConfig{AutoMerge: true})
	t.AppendSeparator()

	type BestCheck struct {
		File  content.VideoFile
		Index int
	}

	for _, row := range rows {
		best := BestCheck{File: videos[0], Index: 0}
		for i, v := range videos {
			if row.BetterThan(v, best.File) {
				best = BestCheck{File: v, Index: i}
			}
		}

		colourize := func(v content.VideoFile, vIndex int) string {
			s := row.Value(v)
			if same {
				return c.Sprintf("<lightBlue>%s</>", s)
			}

			if row.Name == "Duration" {
				diff := math.Abs(v.Duration - videos[0].Duration)
				if diff < 5 {
					return c.Sprintf("<lightBlue>%s</>", s)
				}
				if diff >= 5 && diff < 10 {
					return c.Sprintf("<blue>%s</>", s)
				}
			}

			if row.Name == "Resolution" {
				diffW := math.Abs(float64(v.ResolutionW - videos[0].ResolutionW))
				diffH := math.Abs(float64(v.ResolutionH - videos[0].ResolutionH))
				diff := diffW + diffH
				if diff > 0 && diff < 10 {
					return c.Sprintf("<lightBlue>%s</>", s)
				}
				if diff >= 10 && diff < 20 {
					return c.Sprintf("<blue>%s</>", s)
				}
			}

			if row.Name == "Bitrate" {
				srcBitrate := float64(videos[0].BitRate)
				if srcBitrate > 0 {
					diff := math.Abs(float64(v.BitRate)-srcBitrate) / srcBitrate
					if diff < 0.01 {
						return c.Sprintf("<blue>%s</>", s)
					}
				}
			}

			if best.Index == vIndex || row.Equal(v, best.File) {
				return c.Sprintf("<green>%s</>", s)
			}
			return c.Sprintf("<lightRed>%s</>", s)
		}

		r := table.Row{c.Sprintf("<darkGray>%s</>", row.Name)}
		for i, v := range videos {
			r = append(r, colourize(v, i))
		}
		t.AppendRow(r)
	}

	// Handle audio streams comparison
	maxAudioStreams := 0
	for _, video := range videos {
		if len(video.AudioStreams) > maxAudioStreams {
			maxAudioStreams = len(video.AudioStreams)
		}
	}

	audioSorted := make([][]content.FFProbeStreamAudio, len(videos))
	for i, video := range videos {
		audioSorted[i] = video.AudioStreamsSortedByLanguage()
	}

	for i := 0; i < maxAudioStreams; i++ {
		streams := make([]*content.FFProbeStreamAudio, len(videos))
		for j, sorted := range audioSorted {
			if i < len(sorted) {
				streams[j] = &sorted[i]
			}
		}

		bestStreamIndex := -1
		// Find first non-nil stream to start comparison
		var firstStream *content.FFProbeStreamAudio
		for j, stream := range streams {
			if stream != nil {
				firstStream = stream
				bestStreamIndex = j
				break
			}
		}

		if firstStream != nil {
			for j, stream := range streams {
				if stream != nil {
					if stream.Channels > firstStream.Channels {
						bestStreamIndex = j
					}
				}
			}
		}

		colourize := func(stream *content.FFProbeStreamAudio, streamIndex int) string {
			if stream == nil {
				return ""
			}

			s := fmt.Sprintf("%s %s (%s)", stream.CodecName, stream.ChannelLayout, stream.Language)
			if same {
				return c.Sprintf("<lightBlue>%s</>", s)
			}
			if stream.Language != "eng" {
				return c.Sprintf("<magenta>%s</>", s)
			}
			if bestStreamIndex == streamIndex {
				return c.Sprintf("<green>%s</>", s)
			}
			return c.Sprintf("<lightRed>%s</>", s)
		}

		r := table.Row{c.Sprintf("<darkGray>Audio %d</>", i+1)}
		for j, stream := range streams {
			r = append(r, colourize(stream, j))
		}
		t.AppendRow(r)
	}

	// Handle subtitle streams comparison
	maxSubtitleStreams := 0
	for _, video := range videos {
		if len(video.Subtitles) > maxSubtitleStreams {
			maxSubtitleStreams = len(video.Subtitles)
		}
	}

	subtitlesSorted := make([][]content.FFProbeStreamSubtitle, len(videos))
	for i, video := range videos {
		subtitlesSorted[i] = video.SubtitlesSortedByLanguage()
	}

	for i := 0; i < maxSubtitleStreams; i++ {
		streams := make([]*content.FFProbeStreamSubtitle, len(videos))
		for j, sorted := range subtitlesSorted {
			if i < len(sorted) {
				streams[j] = &sorted[i]
			}
		}

		colourize := func(stream *content.FFProbeStreamSubtitle) string {
			if stream == nil {
				return ""
			}
			s := fmt.Sprintf("%s (%s)", stream.Language, stream.CodecName)
			if same {
				return c.Sprintf("<lightBlue>%s</>", s)
			}
			if stream.Language == "eng" {
				return c.Sprintf("<green>%s</>", s)
			}
			return c.Sprintf("<magenta>%s</>", s)
		}

		r := table.Row{c.Sprintf("<darkGray>Subtitle %d</>", i+1)}
		for _, stream := range streams {
			r = append(r, colourize(stream))
		}
		t.AppendRow(r)
	}

	t.Render()

	// trim trailing newline and indent
	output := buf.String()
	output = output[:len(output)-1]
	_, _ = ktio.IndentWriter{W: os.Stdout, Indent: strings.Repeat(" ", indent)}.Write([]byte(output))
}
