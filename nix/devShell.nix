{ pkgs }:
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

    # Shell script to call from the Makefile
    # to build the driver.
    (import ./driverBuildScript.nix {
      inherit pkgs;
      staticBuild = false;
      # Let go determine these
      GOARCH = "";
      GOOS = "";
      CC = "";
    })
  ]
  ++ lib.optional stdenv.isLinux pcsclite
  ++ lib.optional stdenv.isDarwin pkgs.darwin.apple_sdk.frameworks.PCSC;
}
