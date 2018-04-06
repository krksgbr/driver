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
    buildInputs =
      [ go_1_9
        dep
        # Git is a de facto dependency of dep
        git

        gcc

        # node for tests
        nodejs-8_x

        # for deployment to S3
        awscli
      ]
      ++
      (if stdenv.isLinux then
          [ # glibc replacement for static linking
            musl
            # Building pcsclite
            pkgconfig autoconf automake libtool flex
          ]
        else
          []
      );
}
