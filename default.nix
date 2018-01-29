with import <nixpkgs> {};

stdenv.mkDerivation {
    name = "dividat-driver";
    builder = "${bash}/bin/bash";
    buildInputs = [
      go
      dep

      # node for tests
      nodejs-8_x
    ];
}
