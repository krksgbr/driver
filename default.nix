with (import ./nix/nixpkgs.nix) {};

stdenv.mkDerivation {
    name = "dividat-driver";
    builder = "${bash}/bin/bash";
    buildInputs =
    [ 
        go_1_9
        dep
        # Git is a de facto dependency of dep
        git

        gcc

        # node for tests
        nodejs-8_x

        # for building releases
        openssl upx
        # for deployment to S3
        awscli

      ]
      # PCSC on Darwin
      ++ lib.optional stdenv.isDarwin pkgs.darwin.apple_sdk.frameworks.PCSC
      ++ lib.optional stdenv.isLinux [ pcsclite ];
}
