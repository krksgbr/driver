{ pkgs, darwinCrossBuildScript }:
with pkgs;
mkShell
{
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
    pkg-config

    # Script to cross build on MacOS.
    darwinCrossBuildScript

  ]
  ++ lib.optional stdenv.isLinux pcsclite
  ++ lib.optional stdenv.isDarwin
    # PCSC on Darwin
    # This is only used for development and has nothing to do with `darwinCrossBuildScript`.
    # The script uses the system's clang and PCSC framework, instead of gcc and nix's PCSC.
    # See `nix/crossBuild.nix` for details.
    pkgs.darwin.apple_sdk.frameworks.PCSC
  ;
}
