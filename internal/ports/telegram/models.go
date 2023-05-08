package ports

import (
	"io"
)

type ChannelUpdate struct {
	Username   string
	Title      string
	ID         int64
	IsBotAdded bool
}

type Command struct {
	UserID  int64
	Command string
	Args    []string
}

type Post struct {
	ChannelID int64
	Text      string
	Files     []File
}

type File struct {
	Name   string
	Reader io.Reader
}
