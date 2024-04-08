{
  description = ''
  Provides cross-compiled binaries of Dividat driver for Windows and Linux,
  and a development shell for Linux and macOS.
  '';
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/23.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        version = pkgs.lib.strings.fileContents ./VERSION;
        crossBuild = import ./nix/crossBuild.nix {
          inherit nixpkgs;
          inherit system;
          inherit version;
        };
      in
      {
        packages.dividat-driver = {
          inherit (crossBuild) x86_64-linux;
          inherit (crossBuild) x86_64-windows;
        };
        devShells.default = import ./nix/devShell.nix {
          inherit pkgs;
          inherit (crossBuild) darwinCrossBuildScript;
        };
      }
    );
}

