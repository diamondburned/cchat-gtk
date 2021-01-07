{ pkgs ? import <nixpkgs> {} }:

let nostrip = pkg: pkgs.enableDebugging (pkg.overrideAttrs(old: {
		dontStrip = true;
		doCheck   = false;
		NIX_CFLAGS_COMPILE = (old.NIX_CFLAGS_COMPILE or "") + " -g";
	}));

	libhandy = pkgs.libhandy.overrideAttrs(old: {
		name = "libhandy-1.0.1";
		src  = builtins.fetchGit {
			url = "https://gitlab.gnome.org/GNOME/libhandy.git";
			rev = "5cee0927b8b39dea1b2a62ec6d19169f73ba06c6";
		};
		patches = [];

		buildInputs = old.buildInputs ++ (with pkgs; [
			(nostrip gnome3.librsvg)
			(nostrip gdk-pixbuf)
		]);
	});

in pkgs.stdenv.mkDerivation rec {
	name = "cchat-gtk";
	version = "0.0.2";

	buildInputs = [
		libhandy
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
