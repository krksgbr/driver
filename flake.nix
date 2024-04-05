{
  description = "Cross compile dividat driver";
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/23.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        version = builtins.readFile ./VERSION;
        crossBuildFor = import ./nix/crossBuild.nix {
          inherit nixpkgs;
          inherit system;
          inherit version;
        };
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages.dividat-driver = {
          x86_64-linux = crossBuildFor "x86_64-unknown-linux-musl";
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

