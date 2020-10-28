{ pkgs ? import <nixpkgs> {} }:

pkgs.stdenv.mkDerivation rec {
	name = "cchat-gtk";
	version = "0.0.2";

	buildInputs = with pkgs; [
		libhandy gnome3.gspell gnome3.glib gnome3.gtk
	];

	nativeBuildInputs = with pkgs; [
		pkgconfig go
	];
}
