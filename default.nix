with import <nixpkgs> {};

stdenv.mkDerivation {
    name = "dividat-driver";
    builder = "${bash}/bin/bash";
    buildInputs = [
      go_1_9
      dep

      # node for tests
      nodejs-8_x
    ];
}
