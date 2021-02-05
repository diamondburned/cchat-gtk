BIN ?= cchat-gtk
GO ?= go

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
DATADIR ?= $(PREFIX)/share

cchat-gtk: 
	$(GO) build -v -o bin/$(BIN)
	@strip bin/$(BIN)

install:
	install -Dm775 bin/$(BIN) $(DESTDIR)$(BINDIR)/$(BIN)
	install -Dm775 cchat-gtk.desktop $(DESTDIR)$(DATADIR)/applications/cchat-gtk.desktop
	install -Dm775 icons/cchat.png $(DESTDIR)$(DATADIR)/pixmaps/cchat-gtk.png

uninstall:
	rm -rf $(DESTDIR)$(BINDIR)/$(BIN)
	rm -rf $(DESTDIR)$(DATADIR)/pixmaps/cchat-gtk.png
	rm -rf $(DESTDIR)$(DATADIR)/applications/cchat-gtk.desktop

clean:
	rm -rf bin/

.PHONY: cchat-gtk