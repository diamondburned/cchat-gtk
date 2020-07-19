module github.com/diamondburned/cchat-gtk

go 1.14

replace github.com/gotk3/gotk3 => github.com/diamondburned/gotk3 v0.0.0-20200630065217-97aeb06d705d

replace github.com/diamondburned/cchat-discord => ../cchat-discord/

require (
	github.com/Xuanwo/go-locale v0.2.0
	github.com/alecthomas/chroma v0.7.3
	github.com/diamondburned/cchat v0.0.45
	github.com/diamondburned/cchat-discord v0.0.0-20200718071554-360b34de2a71
	github.com/diamondburned/cchat-mock v0.0.0-20200709231652-ad222ce5a74b
	github.com/diamondburned/imgutil v0.0.0-20200710174014-8a3be144a972
	github.com/disintegration/imaging v1.6.2
	github.com/goodsign/monday v1.0.0
	github.com/gotk3/gotk3 v0.4.1-0.20200524052254-cb2aa31c6194
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/ianlancetaylor/cgosymbolizer v0.0.0-20200424224625-be1b05b0b279
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/twmb/murmur3 v1.1.3
	github.com/zalando/go-keyring v0.0.0-20200121091418-667557018717
	gopkg.in/yaml.v2 v2.2.7 // indirect
)
