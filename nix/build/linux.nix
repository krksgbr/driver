with (import ../nixpkgs.nix) { crossSystem = { config = "x86_64-unknown-linux-gnu"; }; };

stdenv.mkDerivation {
  name = "dividat-driver";

  src = ../../src;

  nativeBuildInputs = [ pkgconfig ];

  buildInputs = [
    pcsclite
  ];

}
