package message

import (
	"context"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat/text"
)

// Author implements cchat.Author. It effectively contains a copy of
// cchat.Author.
type Author struct {
	ID   cchat.ID
	Name rich.NameContainer
}

// NewAuthor creates a new Author that is a copy of the given author.
func NewAuthor(author cchat.User) Author {
	a := Author{ID: author.ID()}
	a.Name.QueueNamer(context.Background(), author)
	return a
}

// NewCustomAuthor creates a new Author from the given parameters.
func NewCustomAuthor(id cchat.ID, name text.Rich) Author {
	return Author{
		ID:   id,
		Name: rich.NameContainer{LabelState: *rich.NewLabelState(name)},
	}
}

// Update sets a new name.
func (author *Author) Update(user cchat.Namer) {
	author.Name.QueueNamer(context.Background(), user)
}
