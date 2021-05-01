{ unstable ? import <unstable> {} }:

unstable.stdenv.mkDerivation rec {
	name = "cchat-gtk";
	version = "0.0.2";

	buildInputs = with unstable; [
		libhandy
		gnome3.gspell
		gnome3.glib
		gnome3.gtk
	];

	nativeBuildInputs = with unstable; [
		pkgconfig
		go
		wrapGAppsHook
	];
}
