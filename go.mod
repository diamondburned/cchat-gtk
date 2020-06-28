module github.com/diamondburned/cchat-gtk

go 1.14

replace github.com/gotk3/gotk3 => github.com/diamondburned/gotk3 v0.0.0-20200619213419-0533bcce0dd6

// replace github.com/diamondburned/cchat-discord => ../cchat-discord/

require (
	github.com/Xuanwo/go-locale v0.2.0
	github.com/alecthomas/chroma v0.7.3
	github.com/diamondburned/cchat v0.0.35
	github.com/diamondburned/cchat-discord v0.0.0-20200619222738-e5babcbb42e3
	github.com/diamondburned/cchat-mock v0.0.0-20200628063912-3155c1b6d6a9
	github.com/diamondburned/imgutil v0.0.0-20200611215339-650ac7cfaf64
	github.com/goodsign/monday v1.0.0
	github.com/google/btree v1.0.0 // indirect
	github.com/gotk3/gotk3 v0.4.1-0.20200524052254-cb2aa31c6194
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/ianlancetaylor/cgosymbolizer v0.0.0-20200424224625-be1b05b0b279
	github.com/markbates/pkger v0.17.0
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/twmb/murmur3 v1.1.3
	github.com/zalando/go-keyring v0.0.0-20200121091418-667557018717
)
