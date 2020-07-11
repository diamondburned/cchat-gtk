## Sidebar refactoring

Maybe put services separately in the left sidebar like so:

![](https://miro.medium.com/max/1600/1*DSH66RN5DA5UQdZ2xE2I-g.png)

## Behavioral changes

Top-level server loads can probably lazy-load, but independent servers can
probably be all loaded at once. This might not be a good idea for guild folders.

cchat-gtk should also store what's expanded into a config. This is pretty
trivial to do.

## Spellcheck

Write a Golang gspell binding and use that.

## Frame limiter

Limit to 25fps if background using glib.TimeoutAdd.
