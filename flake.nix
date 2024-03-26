{
  description = "Cross compile dividat driver";
  inputs = {
    nixpkgs.url =
      # Snapshot from https://github.com/nixos/nixpkgs/tree/nixos-23.05 on 2023 Sep 7
      "github:nixos/nixpkgs/4077a0e4ac3356222bc1f0a070af7939c3098535";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        crossBuildFor = import ./nix/crossBuild.nix {
          inherit nixpkgs;
          inherit system;
        };
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages.dividat-driver = {
          x86_64-linux = crossBuildFor "x86_64-unknown-linux-gnu";
          x86_64-windows = crossBuildFor "x86_64-w64-mingw32";
          aarch64-darwin =
            if pkgs.stdenv.isDarwin then
              crossBuildFor "aarch64-apple-darwin"
            else
              builtins.throw "Building the darwin package is only supported on MacOS.";

        };
        devShells.default = with pkgs;
          mkShell {
            buildInputs = [
              go
              gcc

              # node for tests
              nodejs

              # for building releases
              openssl
              upx

              # for deployment to S3
              awscli

              # Required for building go dependencies
              autoconf
              automake
              libtool
              flex
              pkgconfig

            ] ++ lib.optional stdenv.isDarwin pkgs.darwin.apple_sdk.frameworks.PCSC # PCSC on Darwin
            ++ lib.optional stdenv.isLinux pcsclite;

            # GOPATH is set to a readonly directory
            # This seems to be fixed with nixpkgs 20.03
            # https://github.com/NixOS/nixpkgs/issues/90136
            shellHook = ''
              export GOPATH="$HOME/.go"
            '';

          };
      }
    );
}

