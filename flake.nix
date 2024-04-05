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
        devShells.default = import ./nix/devShell.nix { inherit pkgs; };
      }
    );
}

