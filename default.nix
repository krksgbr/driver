with import <nixpkgs> {
  overlays = [ 
    (self: super: {
      dep = import ./nix/dep.nix super;
    })
  ];
};



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
