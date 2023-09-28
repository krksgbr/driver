with (import ../nixpkgs.nix) { crossSystem = { config = "x86_64-unknown-linux-musl"; }; };

stdenv.mkDerivation {
  name = "dividat-driver";

  src = ../../src;

  nativeBuildInputs = [ pkgconfig ];

  buildInputs = [
    (import ./../pcsclite {inherit lib stdenv fetchFromGitHub pkgconfig autoconf automake libtool flex python3 perl;})
  ];

}
