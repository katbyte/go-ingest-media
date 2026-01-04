package cli

import (
	"bytes"
	"fmt"
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

func RenderVideoComparisonTable(indent int, srcVideo content.VideoFile, dstVideos []content.VideoFile, srcIndex int) {
	var buf bytes.Buffer
	t := table.NewWriter()
	t.SetOutputMirror(&buf)
	t.SetStyle(tablestyle)

	same := true
	for _, dstVideo := range dstVideos {
		same = same && srcVideo.IsBasicallyTheSameTo(dstVideo)
	}

	srcHeader := "Source"
	if srcIndex > 0 {
		srcHeader = fmt.Sprintf("Source %d", srcIndex)
	}
	header := table.Row{"", srcHeader}
	for i := range dstVideos {
		header = append(header, fmt.Sprintf("Destination %d", i+1))
	}

	t.AppendHeader(header, table.RowConfig{AutoMerge: true})
	t.AppendSeparator()

	type BestCheck struct {
		File  content.VideoFile
		Index int
	}

	for _, row := range rows {
		best := BestCheck{File: srcVideo, Index: -1}
		for i, dstVideo := range dstVideos {
			if row.BetterThan(dstVideo, best.File) {
				best = BestCheck{File: dstVideo, Index: i}
			}
		}

		colourize := func(v content.VideoFile, vIndex int) string {
			s := row.Value(v)
			if same {
				return c.Sprintf("<lightBlue>%s</>", s)
			}
			if best.Index == vIndex {
				return c.Sprintf("<green>%s</>", s)
			}
			return c.Sprintf("<lightRed>%s</>", s)
		}

		r := table.Row{c.Sprintf("<darkGray>%s</>", row.Name), colourize(srcVideo, -1)}
		for i, dstVideo := range dstVideos {
			r = append(r, colourize(dstVideo, i))
		}
		t.AppendRow(r)
	}

	// Handle audio streams comparison
	maxAudioStreams := len(srcVideo.AudioStreams)
	for _, dstVideo := range dstVideos {
		if len(dstVideo.AudioStreams) > maxAudioStreams {
			maxAudioStreams = len(dstVideo.AudioStreams)
		}
	}

	srcAudioSorted := srcVideo.AudioStreamsSortedByLanguage()
	dstAudioSorted := make([][]content.FFProbeStreamAudio, len(dstVideos))
	for i, dstVideo := range dstVideos {
		dstAudioSorted[i] = dstVideo.AudioStreamsSortedByLanguage()
	}

	for i := 0; i < maxAudioStreams; i++ {
		var srcStream *content.FFProbeStreamAudio
		if i < len(srcAudioSorted) {
			srcStream = &srcAudioSorted[i]
		}

		dstStreams := make([]*content.FFProbeStreamAudio, len(dstVideos))
		for j, sorted := range dstAudioSorted {
			if i < len(sorted) {
				dstStreams[j] = &sorted[i]
			}
		}

		bestStreamIndex := -1 // -1 for src
		if srcStream != nil {
			for j, dstStream := range dstStreams {
				if dstStream != nil {
					// simple more channels is better
					if dstStream.Channels > srcStream.Channels {
						bestStreamIndex = j
					}
				}
			}
		} else {
			// src stream is nil, find first non-nil dst stream
			for j, dstStream := range dstStreams {
				if dstStream != nil {
					bestStreamIndex = j
					break
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

		r := table.Row{c.Sprintf("<darkGray>Audio %d</>", i+1), colourize(srcStream, -1)}
		for j, dstStream := range dstStreams {
			r = append(r, colourize(dstStream, j))
		}
		t.AppendRow(r)
	}

	// Handle subtitle streams comparison
	maxSubtitleStreams := len(srcVideo.Subtitles)
	for _, dstVideo := range dstVideos {
		if len(dstVideo.Subtitles) > maxSubtitleStreams {
			maxSubtitleStreams = len(dstVideo.Subtitles)
		}
	}

	srcSubtitles := srcVideo.SubtitlesSortedByLanguage()
	dstSubtitlesSorted := make([][]content.FFProbeStreamSubtitle, len(dstVideos))
	for i, dstVideo := range dstVideos {
		dstSubtitlesSorted[i] = dstVideo.SubtitlesSortedByLanguage()
	}

	for i := 0; i < maxSubtitleStreams; i++ {
		var srcStream *content.FFProbeStreamSubtitle
		if i < len(srcSubtitles) {
			srcStream = &srcSubtitles[i]
		}

		dstStreams := make([]*content.FFProbeStreamSubtitle, len(dstVideos))
		for j, sorted := range dstSubtitlesSorted {
			if i < len(sorted) {
				dstStreams[j] = &sorted[i]
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

		r := table.Row{c.Sprintf("<darkGray>Subtitle %d</>", i+1), colourize(srcStream)}
		for _, dstStream := range dstStreams {
			r = append(r, colourize(dstStream))
		}
		t.AppendRow(r)
	}

	t.Render()

	// trim trailing newline and indent
	output := buf.String()
	output = output[:len(output)-1]
	_, _ = ktio.IndentWriter{W: os.Stdout, Indent: strings.Repeat(" ", indent)}.Write([]byte(output))
}
