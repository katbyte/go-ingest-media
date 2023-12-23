package cli

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	c "github.com/gookit/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/katbyte/go-ingest-media/lib/content"
	"github.com/katbyte/go-ingest-media/lib/ktio"
	_ "github.com/mattn/go-sqlite3"
)

var tableIndent = "  "
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

var videoTableRows = []TableRow{
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
func RenderVideoComparisonTable(srcVideo content.VideoFile, dstVideos []content.VideoFile) {

	if len(dstVideos) > 1 {
		panic("multiple destination videos not supported yet")
	}
	dstVideo := dstVideos[0] // we dont' support multiple video files yet

	same := srcVideo.IsBasicallyTheSameTo(dstVideo)

	/*tracks := len (m.SrcVideo.AudioStreams)
	if len(dstVideo.AudioStreams) > tracks {
		tracks = len(dstVideo.AudioStreams)
	}

	//audio track information
	audio := make([]string, tracks)
	for i, a := range m.SrcVideo.AudioStreams {

	}*/

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
	for _, row := range videoTableRows {
		n := c.Sprintf("<darkGray>" + row.Name + "</>")
		if row.Equal(srcVideo, dstVideo) {
			t.AppendRow(table.Row{n, c.Sprintf("<lightBlue>" + row.Value(srcVideo) + "</>"), c.Sprintf("<lightBlue>" + row.Value(dstVideo) + "</>")})
		} else if row.BetterThan(srcVideo, dstVideo) {
			t.AppendRow(table.Row{n, c.Sprintf("<red>" + row.Value(srcVideo) + "</>"), c.Sprintf("<green>" + row.Value(dstVideo) + "</>")})
		} else {
			t.AppendRow(table.Row{n, c.Sprintf("<green>" + row.Value(srcVideo) + "</>"), c.Sprintf("<red>" + row.Value(dstVideo) + "</>")})
		}
	}

	t.Render()

	// trim trailing newline and indent
	output := buf.String()
	output = output[:len(output)-1]
	ktio.IndentWriter{W: os.Stdout, Indent: tableIndent}.Write([]byte(output))
}
