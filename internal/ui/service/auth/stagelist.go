package auth

import (
	"github.com/diamondburned/cchat"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

type StageList struct {
	*gtk.ScrolledWindow
	ListBox *gtk.ListBox
}

func NewStageList(authers []cchat.Authenticator, fn func(cchat.Authenticator)) *StageList {
	list, _ := gtk.ListBoxNew()
	list.SetSelectionMode(gtk.SELECTION_BROWSE)
	list.SetActivateOnSingleClick(true)
	list.Connect("row-activated", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		fn(authers[row.GetIndex()])
	})
	list.Show()

	for _, auth := range authers {
		row := handy.ActionRowNew()
		row.SetActivatable(true)
		row.SetTitle(auth.Name().String())
		row.SetSubtitle(auth.Description().String())
		row.Show()

		list.Add(row)
	}

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	sw.SetSizeRequest(200, 0)
	sw.SetHAlign(gtk.ALIGN_FILL)
	sw.SetHExpand(false)
	sw.Add(list)

	return &StageList{
		ScrolledWindow: sw,
		ListBox:        list,
	}
}

func (slist *StageList) SelectFirst() {
	if first := slist.ListBox.GetRowAtIndex(0); first != nil {
		first.Activate()
	}
}
