{ pkgs ? import <nixpkgs> {} }:

let libhandy = pkgs.libhandy.overrideAttrs(old: {
	name = "libhandy-1.0.1";
	src  = builtins.fetchGit {
		url = "https://gitlab.gnome.org/GNOME/libhandy.git";
		rev = "5cee0927b8b39dea1b2a62ec6d19169f73ba06c6";
	};
	patches = [];

	buildInputs = old.buildInputs ++ (with pkgs; [
		gnome3.librsvg
		gdk-pixbuf
	]);
});

in pkgs.stdenv.mkDerivation rec {
	name = "cchat-gtk";
	version = "0.0.2";

	buildInputs = [ libhandy ] ++ (with pkgs; [
		gnome3.gspell gnome3.glib gnome3.gtk
	]);

	nativeBuildInputs = with pkgs; [
		pkgconfig go
	];
}
