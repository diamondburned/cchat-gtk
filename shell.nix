{ pkgs ? import <nixpkgs> {} }:

pkgs.stdenv.mkDerivation rec {
	name = "cchat-gtk";
	version = "0.0.2";

	buildInputs = [
		pkgs.libhandy
		pkgs.gnome3.gspell
		pkgs.gnome3.glib
		pkgs.gnome3.gtk
	];

	nativeBuildInputs = with pkgs; [
		pkgconfig go
	];

	# Debug flags.
	CGO_CFLAGS   = "-g";
	CGO_CXXFLAGS = "-g";
}
