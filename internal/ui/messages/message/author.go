package message

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat/text"
)

// Author implements cchat.Author. It effectively contains a copy of
// cchat.Author.
type Author struct {
	id        cchat.ID
	name      text.Rich
	avatarURL string
}

var _ cchat.Author = (*Author)(nil)

// NewAuthor creates a new Author that is a copy of the given author.
func NewAuthor(author cchat.Author) Author {
	a := Author{}
	a.Update(author)
	return a
}

// NewCustomAuthor creates a new Author from the given parameters.
func NewCustomAuthor(id cchat.ID, name text.Rich, avatar string) Author {
	return Author{
		id,
		name,
		avatar,
	}
}

func (a *Author) Update(author cchat.Author) {
	a.id = author.ID()
	a.name = author.Name()
	a.avatarURL = author.Avatar()
}

func (a Author) ID() string {
	return a.id
}

func (a Author) Name() text.Rich {
	return a.name
}

func (a Author) Avatar() string {
	return a.avatarURL
}
