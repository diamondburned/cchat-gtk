module github.com/diamondburned/cchat-gtk

go 1.14

replace github.com/diamondburned/cchat-mock => ../cchat-mock/

require (
	github.com/Xuanwo/go-locale v0.2.0
	github.com/diamondburned/cchat v0.0.15
	github.com/diamondburned/cchat-mock v0.0.0-20200605224934-31a53c555ea2
	github.com/diamondburned/imgutil v0.0.0-20200606035324-63abbc0fdea6
	github.com/die-net/lrucache v0.0.0-20190707192454-883874fe3947
	github.com/goodsign/monday v1.0.0
	github.com/google/btree v1.0.0 // indirect
	github.com/gotk3/gotk3 v0.4.1-0.20200524052254-cb2aa31c6194
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79
	github.com/markbates/pkger v0.17.0
	github.com/peterbourgon/diskv v2.0.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/zalando/go-keyring v0.0.0-20200121091418-667557018717
)
