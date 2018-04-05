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
      # Git is a de facto dependency of dep
      git

      gcc
      # Required for static linking on Linux
      (if stdenv.isDarwin then null else musl)

      # node for tests
      nodejs-8_x

      # for deployment to S3
      awscli
    ];
}
