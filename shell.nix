{ pkgs ? import <nixpkgs> {} }:

pkgs.stdenv.mkDerivation rec {
	name = "cchat-gtk";
	version = "0.0.2";

	buildInputs = with pkgs; [
		gnome3.glib gnome3.gtk # libhandy
	];

	nativeBuildInputs = with pkgs; [
		pkgconfig go
	];
}
