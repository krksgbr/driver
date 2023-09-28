with (import ../nixpkgs.nix) { crossSystem = { config = "x86_64-unknown-linux-musl"; }; };

stdenv.mkDerivation {
  name = "dividat-driver";

  src = ../../src;

  nativeBuildInputs = [ pkgconfig ];

  buildInputs = [
    (import ./../pcsclite {
      inherit lib stdenv fetchFromGitHub  pkg-config autoconf-archive autoreconfHook python3 perl;
    })
  ];

}
