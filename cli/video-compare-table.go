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
	{"Ext",
		func(file content.VideoFile) string { return file.Ext },
		func(v1, v2 content.VideoFile) bool { return v1.Ext == v2.Ext },
		func(v1, v2 content.VideoFile) bool {
			return content.VideoExtensionIndex(v1.Ext) > content.VideoExtensionIndex(v2.Ext)
		},
	},
	{"Size",
		func(file content.VideoFile) string { return fmt.Sprintf("%0.2f", file.SizeGb) },
		func(v1, v2 content.VideoFile) bool { return v1.SizeGb == v2.SizeGb },
		func(v1, v2 content.VideoFile) bool { return v1.SizeGb < v2.SizeGb },
	},
	{"Resolution",
		func(file content.VideoFile) string { return file.Resolution },
		func(v1, v2 content.VideoFile) bool { return v1.Resolution == v2.Resolution },
		func(v1, v2 content.VideoFile) bool {
			return v1.ResolutionW*v1.ResolutionH > v2.ResolutionW*v2.ResolutionH
		},
	},
	{"Codec",
		func(file content.VideoFile) string { return file.VideoStream.CodecName },
		func(v1, v2 content.VideoFile) bool { return v1.VideoStream.CodecName == v2.VideoStream.CodecName },
		func(v1, v2 content.VideoFile) bool { return v1.VideoStream.CodecName > v2.VideoStream.CodecName },
	},
	{"Profile",
		func(file content.VideoFile) string { return file.VideoStream.Profile },
		func(v1, v2 content.VideoFile) bool { return v1.VideoStream.Profile == v2.VideoStream.Profile },
		func(v1, v2 content.VideoFile) bool { return v1.VideoStream.Profile > v2.VideoStream.Profile },
	},
	{"Duration",
		func(file content.VideoFile) string { return fmt.Sprintf("%0.2f", file.Duration) },
		func(v1, v2 content.VideoFile) bool { return v1.Duration == v2.Duration },
		func(v1, v2 content.VideoFile) bool { return v1.Duration > v2.Duration },
	},
	{"Bitrate",
		func(file content.VideoFile) string { return strconv.Itoa(file.BitRate) },
		func(v1, v2 content.VideoFile) bool { return v1.BitRate == v2.BitRate },
		func(v1, v2 content.VideoFile) bool { return v1.BitRate > v2.BitRate },
	},
}

// todo change to accept array of videos and compare them
func RenderVideoComparisonTable(srcVideo content.VideoFile, dstVideos []content.VideoFile, indent int) {

	if len(dstVideos) > 1 {
		panic("multiple destination videos not supported yet")
	}
	dstVideo := dstVideos[0] // we dont' support multiple video files yet

	same := srcVideo.IsBasicallyTheSameTo(dstVideo)

	var buf bytes.Buffer
	t := table.NewWriter()
	t.SetOutputMirror(&buf)
	t.SetStyle(tablestyle)
	if same {
		t.AppendHeader(table.Row{"", c.Sprintf("<green>Source</>"), c.Sprintf("<green>Destination</>")})
	} else {
		t.AppendHeader(table.Row{"", c.Sprintf("<white>Source</>"), c.Sprintf("<white>Destination</>")})
	}
	t.AppendSeparator()

	for _, row := range rows {
		n := c.Sprintf("<darkGray>" + row.Name + "</>")
		if row.Equal(srcVideo, dstVideo) {
			t.AppendRow(table.Row{n, c.Sprintf("<lightBlue>" + row.Value(srcVideo) + "</>"), c.Sprintf("<lightBlue>" + row.Value(dstVideo) + "</>")})
		} else if row.BetterThan(srcVideo, dstVideo) {
			t.AppendRow(table.Row{n, c.Sprintf("<green>" + row.Value(srcVideo) + "</>"), c.Sprintf("<red>" + row.Value(dstVideo) + "</>")})
		} else {
			t.AppendRow(table.Row{n, c.Sprintf("<red>" + row.Value(srcVideo) + "</>"), c.Sprintf("<green>" + row.Value(dstVideo) + "</>")})
		}
	}

	// audio rows
	maxAudioStreams := len(srcVideo.AudioStreams)
	if len(dstVideos[0].AudioStreams) > maxAudioStreams {
		maxAudioStreams = len(dstVideos[0].AudioStreams)
	}

	srcAudioSorted := srcVideo.AudioStreamsSortedByLanguage()
	dstAudioSorted := dstVideo.AudioStreamsSortedByLanguage()

	for i := 0; i < maxAudioStreams; i++ {
		var srcStream, dstStream string
		if i < len(srcAudioSorted) {
			srcStream = fmt.Sprintf("%s (%s)", srcAudioSorted[i].Language, srcAudioSorted[i].CodecName)
		}
		if i < len(dstAudioSorted) {
			dstStream = fmt.Sprintf("%s (%s)", dstAudioSorted[i].Language, dstAudioSorted[i].CodecName)
		}

		srcColour := "lightBlue"
		dstColour := "lightBlue"

		if i < len(srcAudioSorted) && i < len(dstAudioSorted) {
			if srcAudioSorted[i].Language == dstAudioSorted[i].Language {
				if srcAudioSorted[i].CodecName == dstAudioSorted[i].CodecName {
					srcColour = "lightBlue"
					dstColour = "lightBlue"
				} else {
					srcColour = "magenta"
					dstColour = "magenta"
				}
			} else {
				if srcAudioSorted[i].Language == "eng" {
					srcColour = "green"
					dstColour = "red"
				} else {
					srcColour = "magenta"
					dstColour = "magenta"
				}
			}
		} else if i < len(srcAudioSorted) {
			srcColour = "red"
		} else if i < len(dstAudioSorted) {
			dstColour = "green"
		}

		t.AppendRow(table.Row{
			c.Sprintf("<darkGray>Audio %d</>", i+1),
			c.Sprintf("<" + srcColour + ">" + srcStream + "</>"),
			c.Sprintf("<" + dstColour + ">" + dstStream + "</>"),
		})
	}

	// Handle subtitle streams
	maxSubtitleStreams := len(srcVideo.Subtitles)
	if len(dstVideo.Subtitles) > maxSubtitleStreams {
		maxSubtitleStreams = len(dstVideo.Subtitles)
	}

	srcSubtitles := srcVideo.SubtitlesSortedByLanguage()
	dstSubtitles := dstVideo.SubtitlesSortedByLanguage()

	for i := 0; i < maxSubtitleStreams; i++ {
		var srcStream, dstStream string

		if i < len(srcSubtitles) {
			srcStream = fmt.Sprintf("%s (%s)", srcSubtitles[i].Language, srcSubtitles[i].CodecName)
		}
		if i < len(dstSubtitles) {
			dstStream = fmt.Sprintf("%s (%s)", dstSubtitles[i].Language, dstSubtitles[i].CodecName)
		}

		srcColour := "lightBlue"
		dstColour := "lightBlue"

		if i < len(srcSubtitles) && i < len(dstSubtitles) {
			if srcSubtitles[i].Language == dstSubtitles[i].Language {
				srcColour = "lightBlue"
				dstColour = "lightBlue"
			} else {
				if srcSubtitles[i].Language == "eng" {
					srcColour = "green"
					dstColour = "red"
				} else {
					srcColour = "magenta"
					dstColour = "magenta"
				}
			}
		} else if i < len(srcSubtitles) {
			srcColour = "red"
		} else if i < len(dstSubtitles) {
			dstColour = "green"
		}

		t.AppendRow(table.Row{
			c.Sprintf("<darkGray>Subtitle %d</>", i+1),
			c.Sprintf("<" + srcColour + ">" + srcStream + "</>"),
			c.Sprintf("<" + dstColour + ">" + dstStream + "</>"),
		})
	}

	t.Render()

	// trim trailing newline and indent
	output := buf.String()
	output = output[:len(output)-1]
	ktio.IndentWriter{W: os.Stdout, Indent: strings.Repeat(" ", indent)}.Write([]byte(output))
}
