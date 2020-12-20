module github.com/diamondburned/cchat-gtk

go 1.14

replace github.com/gotk3/gotk3 => github.com/diamondburned/gotk3 v0.0.0-20201209182406-e7291341a091

//replace github.com/diamondburned/cchat-discord => ../cchat-discord

//replace github.com/diamondburned/ningen/v2 => ../../ningen

//replace github.com/diamondburned/arikawa/v2 => ../../arikawa

require (
	github.com/Xuanwo/go-locale v1.0.0
	github.com/alecthomas/chroma v0.7.3
	github.com/diamondburned/cchat v0.3.15
	github.com/diamondburned/cchat-discord v0.0.0-20201220081640-288591a535af
	github.com/diamondburned/cchat-mock v0.0.0-20201115033644-df8d1b10f9db
	github.com/diamondburned/gspell v0.0.0-20200830182722-77e5d27d6894
	github.com/diamondburned/handy v0.0.0-20200829011954-4667e7a918f4
	github.com/diamondburned/imgutil v0.0.0-20200710174014-8a3be144a972
	github.com/disintegration/imaging v1.6.2
	github.com/goodsign/monday v1.0.0
	github.com/gotk3/gotk3 v0.4.1-0.20200524052254-cb2aa31c6194
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/twmb/murmur3 v1.1.3
	github.com/zalando/go-keyring v0.0.0-20200121091418-667557018717
)
