package memberlist

import (
	"context"
	"fmt"

	"github.com/diamondburned/cchat"
	"github.com/diamondburned/cchat-gtk/internal/gts"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives"
	"github.com/diamondburned/cchat-gtk/internal/ui/primitives/roundimage"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/labeluri"
	"github.com/diamondburned/cchat-gtk/internal/ui/rich/parser/markup"
	"github.com/diamondburned/cchat/text"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/pkg/errors"
)

var MemberListWidth = 250

type Controller interface {
	MemberListUpdated(c *Container)
}

type Container struct {
	*gtk.Revealer
	Scroll *gtk.ScrolledWindow
	Main   *gtk.Box
	ctrl   Controller

	// states

	// map id -> *Section
	Sections map[string]*Section
	stop     func()

	eventQueue eventQueue
}

var memberListCSS = primitives.PrepareClassCSS("member-list", `
	.member-list {
		background-color: @theme_base_color;
	}
`)

func New(ctrl Controller) *Container {
	main, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	main.SetSizeRequest(250, -1)
	main.Show()
	memberListCSS(main)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	sw.Add(main)
	sw.Show()

	rev, _ := gtk.RevealerNew()
	rev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_LEFT)
	rev.SetTransitionDuration(75)
	rev.SetRevealChild(false)
	rev.Add(sw)

	return &Container{
		Revealer: rev,
		Scroll:   sw,
		Main:     main,
		ctrl:     ctrl,
		Sections: map[string]*Section{},
	}
}

// IsEmpty returns whether or not the member view container is empty.
func (c *Container) IsEmpty() bool {
	return len(c.Sections) == 0
}

// Reset removes all old sections.
func (c *Container) Reset() {
	if c.stop != nil {
		c.stop()
		c.stop = nil
	}

	c.Revealer.SetRevealChild(false)
	c.Sections = map[string]*Section{}
}

// TryAsyncList tries to set the member list from the given server. It does type
// assertions and handles asynchronicity. Reset must be called before this.
func (c *Container) TryAsyncList(server cchat.Messenger) {
	ls := server.AsMemberLister()
	if ls == nil {
		return
	}

	gts.Async(func() (func(), error) {
		f, err := ls.ListMembers(context.Background(), c)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to list members")
		}

		return func() { c.stop = f }, nil
	})
}

func (c *Container) SetSections(sections []cchat.MemberSection) {
	c.eventQueue.Add(func() { c.SetSectionsUnsafe(sections) })
}

func (c *Container) SetMember(sectionID string, member cchat.ListMember) {
	c.eventQueue.Add(func() { c.SetMemberUnsafe(sectionID, member) })
}

func (c *Container) RemoveMember(sectionID string, id string) {
	c.eventQueue.Add(func() { c.RemoveMemberUnsafe(sectionID, id) })
}

type sectionInsert struct {
	front    *Section
	sections []*Section
}

func (c *Container) SetSectionsUnsafe(sections []cchat.MemberSection) {
	// Lazily invalidate the container. We could delegate removing old sections
	// to this function instead of Reset to not halt for too long.
	primitives.RemoveChildren(c.Main)

	newSections := make([]*Section, len(sections))
	oldSections := c.Sections

	for i, section := range sections {
		sc, ok := c.Sections[section.ID()]
		if !ok {
			sc = NewSection(section, &c.eventQueue)
		} else {
			sc.Update(section)
		}

		newSections[i] = sc
	}

	// Remove all old sections.

	for id := range c.Sections {
		delete(c.Sections, id)
	}

	// Insert new sections.
	for _, section := range newSections {
		c.Main.Add(section)
		c.Sections[section.ID] = section
	}

	// Destroy old sections.
	for _, section := range oldSections {
		_, notOld := c.Sections[section.ID]
		if notOld {
			continue
		}

		section.Destroy()
	}

	c.ctrl.MemberListUpdated(c)
}

func (c *Container) SetMemberUnsafe(sectionID string, member cchat.ListMember) {
	if s, ok := c.Sections[sectionID]; ok {
		s.SetMember(member)
	}
}

func (c *Container) RemoveMemberUnsafe(sectionID string, id string) {
	if s, ok := c.Sections[sectionID]; ok {
		s.RemoveMember(id)
	}
}

type Section struct {
	*gtk.Box

	ID string

	// state
	name  rich.NameContainer
	total int

	Header *rich.Label
	Body   *gtk.ListBox

	// map id -> *Member
	Members map[string]*Member
}

var sectionHeaderCSS = primitives.PrepareClassCSS("section-header", `
	.section-header {
		margin: 8px 12px;
		margin-bottom: 2px;
	}
`)

var sectionBodyCSS = primitives.PrepareClassCSS("section-body", `
	.section-body {
		background: inherit;
	}
`)

func NewSection(sect cchat.MemberSection, evq EventQueuer) *Section {
	section := &Section{
		ID:   sect.ID(),
		name: rich.NameContainer{},
	}

	section.Header = rich.NewLabel(&section.name)
	section.Header.Show()
	sectionHeaderCSS(section.Header)

	section.Header.SetRenderer(func(rich text.Rich) markup.RenderOutput {
		out := markup.RenderCmplx(rich)
		if section.total > 0 {
			out.Markup += fmt.Sprintf("—%d", section.total)
		}

		return out
	})

	section.Body, _ = gtk.ListBoxNew()
	section.Body.SetSelectionMode(gtk.SELECTION_NONE)
	section.Body.SetActivateOnSingleClick(true)
	section.Body.SetSortFunc(listSortNameAsc) // A-Z
	section.Body.Show()
	sectionBodyCSS(section.Body)

	section.Box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	section.Box.PackStart(section.Header, false, false, 0)
	section.Box.PackStart(section.Body, false, false, 0)
	section.Box.Show()

	members := map[string]*Member{}

	// On row click, show the mention popup if any.
	section.Body.Connect("row-activated", func(_ *gtk.ListBox, r *gtk.ListBoxRow) {
		i := r.GetIndex()

		// Cold path; we can afford searching in the map.
		for _, member := range members {
			if member.ListBoxRow.GetIndex() == i {
				member.Popup(evq)
			}
		}
	})

	section.name.QueueNamer(context.Background(), sect)
	section.Header.Connect("destroy", section.name.Stop)
	section.Members = members

	return section
}

func (s *Section) Destroy() {
	s.name.Stop()
	s.Box.Destroy()
}

func (s *Section) Update(sect cchat.MemberSection) {
	s.total = sect.Total()
	s.name.QueueNamer(context.Background(), sect)
}

func (s *Section) SetMember(member cchat.ListMember) {
	if m, ok := s.Members[member.ID()]; ok {
		m.Update(member)
		return
	}

	m := NewMember(member)
	m.Show()

	s.Members[member.ID()] = m
	s.Body.Add(m)
}

func (s *Section) RemoveMember(id string) {
	if member, ok := s.Members[id]; ok {
		s.Body.Remove(member)
		delete(s.Members, id)
	}
}

func listSortNameAsc(r1, r2 *gtk.ListBoxRow) int {
	n1, _ := r1.GetName()
	n2, _ := r2.GetName()

	switch {
	case n1 < n2:
		return -1
	case n1 > n2:
		return 1
	default:
		return 0
	}
}

type Member struct {
	*gtk.ListBoxRow
	Main *gtk.Box

	Avatar *roundimage.StillImage
	Name   *rich.Label

	name   rich.LabelState
	second text.Rich
	status cchat.Status
	parent *gtk.ListBox
}

const AvatarSize = 32

var memberRowCSS = primitives.PrepareClassCSS("member-row", `
	.member-row {
		min-height: 42px;
	}
`)

var memberBoxCSS = primitives.PrepareClassCSS("member-box", `
	.member-box {
		margin: 3px 10px;
	}
`)

var avatarBoxMemberCSS = primitives.PrepareClassCSS("avatar-box-member", `
	.avatar-box-member {
		margin-right: 10px;
		padding: 2px;
		border: 1.5px solid;
		border-color: #747F8D; /* Offline Grey */
		border-radius: 99px;
	}

	.avatar-box-member.online {
		border-color: #43B581;
	}
	
	.avatar-box-member.busy {
		border-color: #F04747;
	}
	
	.avatar-box-member.idle {
		border-color: #FAA61A;
	}
`)

var labelMemberCSS = primitives.PrepareClassCSS("label-member", ``)

func NewMember(member cchat.ListMember) *Member {
	m := Member{}

	evb, _ := gtk.EventBoxNew()
	evb.AddEvents(int(gdk.EVENT_ENTER_NOTIFY) | int(gdk.EVENT_LEAVE_NOTIFY))
	evb.Show()

	m.Avatar = roundimage.NewStillImage(evb, 0)
	m.Avatar.SetSize(AvatarSize)
	m.Avatar.SetPlaceholderIcon("user-info-symbolic", AvatarSize)
	m.Avatar.Show()
	rich.BindRoundImage(m.Avatar, &m.name, true)

	avaBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	avaBox.SetVAlign(gtk.ALIGN_CENTER)
	avaBox.PackStart(m.Avatar, false, false, 0)
	avaBox.Show()
	avatarBoxMemberCSS(avaBox)

	m.Name = rich.NewLabel(&m.name)
	m.Name.SetUseMarkup(true)
	m.Name.SetXAlign(0)
	m.Name.SetEllipsize(pango.ELLIPSIZE_END)
	m.Name.Show()
	labelMemberCSS(m.Name)

	// Keep track of the current status class to replace.
	var statusClass string
	styler, _ := avaBox.GetStyleContext()

	m.Name.SetRenderer(func(rich text.Rich) markup.RenderOutput {
		out := markup.RenderCmplxWithConfig(rich, markup.RenderConfig{
			NoMentionLinks: true,
		})

		if statusClass != "" {
			styler.RemoveClass(statusClass)
		}

		statusClass = statusClassName(m.status)
		styler.AddClass(statusClass)

		if !m.second.IsEmpty() {
			out.Markup += fmt.Sprintf(
				`<span alpha="85%%" size="small">`+"\n"+`%s</span>`,
				markup.Render(m.second),
			)
		}

		return out
	})

	m.Main, _ = gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	m.Main.PackStart(avaBox, false, false, 0)
	m.Main.PackStart(m.Name, true, true, 0)
	m.Main.Show()
	memberBoxCSS(m.Main)

	evb.Add(m.Main)

	m.ListBoxRow, _ = gtk.ListBoxRowNew()
	m.ListBoxRow.Add(evb)
	memberRowCSS(m.ListBoxRow)

	m.Update(member)

	return &m
}

func statusClassName(status cchat.Status) string {
	switch status {
	case cchat.StatusOnline:
		return "online"
	case cchat.StatusBusy:
		return "busy"
	case cchat.StatusAway:
		fallthrough
	case cchat.StatusIdle:
		return "idle"
	case cchat.StatusInvisible:
		fallthrough
	case cchat.StatusOffline:
		fallthrough
	default:
		return ""
	}
}

var noMentionLinks = markup.RenderConfig{
	NoMentionLinks: true,
	NoReferencing:  true,
}

func (m *Member) Update(member cchat.ListMember) {
	m.status = member.Status()
	m.second = member.Secondary()

	m.name.SetLabel(member.Name())
	m.ListBoxRow.SetName(member.Name().Content)
}

// Popup pops up the mention popover if any.
func (m *Member) Popup(evq EventQueuer) {
	out := m.Name.Output()

	if len(out.Mentions) == 0 {
		return
	}

	p := labeluri.NewPopoverMentioner(m, out.Input, out.Mentions[0])
	if p == nil {
		return
	}

	// Unbounded concurrency is kind of bad. We should deal with
	// this in the future.
	evq.Activate()
	p.Connect("closed", func(interface{}) { evq.Deactivate() })

	p.SetPosition(gtk.POS_LEFT)
	p.Popup()
}

func statusColors(status cchat.Status) uint32 {
	switch status {
	case cchat.StatusOnline:
		return 0x43B581
	case cchat.StatusBusy:
		return 0xF04747
	case cchat.StatusIdle:
		return 0xFAA61A
	case cchat.StatusOffline:
		fallthrough
	default:
		return 0x747F8D
	}
}
