package content

// adds to the content type (folder) by adding singular video details as 1 movie has 1 video file

type Movie struct {
	Content

	// video details
	Resolution string
	Codec      string
	Audio      string
}

func (l Library) MovieFor(folder string) (*Movie, error) {
	m := Movie{}

	c, err := l.ContentFor(folder)
	if err != nil {
		return nil, err
	}
	m.Content = *c

	return &m, nil
}
