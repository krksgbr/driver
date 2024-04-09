{ pkgs }:
let
  mkCrossBuildShell =
    { GOOS
    , GOARCH
    , CC
    , buildInputs ? [ ]
    , nativeBuildInputs ? [ ]
    , staticBuild
    }:
    pkgs.mkShell {
      inherit nativeBuildInputs;
      buildInputs = [
        pkgs.go
        (import ./driverBuildScript.nix {
          inherit pkgs GOOS GOARCH CC staticBuild;
        })
      ] ++ buildInputs;
    };

in
{
  x86_64-windows =
    let
      mingwPkgs = pkgs.pkgsCross.mingwW64;
      cc = mingwPkgs.stdenv.cc;
    in
    mkCrossBuildShell {
      GOOS = "windows";
      GOARCH = "amd64";
      CC = "${cc}/bin/x86_64-w64-mingw32-gcc";
      buildInputs = [
        cc
        mingwPkgs.windows.mingw_w64_pthreads
      ];
      staticBuild = true;
    };

  x86_64-linux =
    let
      muslPkgs = pkgs.pkgsCross.musl64;
      cc = muslPkgs.gcc;
      static-pcsclite = muslPkgs.pcsclite.overrideAttrs (attrs: {
        configureFlags = attrs.configureFlags ++ [ "--enable-static" "--disable-shared" ];
      });
    in
    mkCrossBuildShell {
      GOOS = "linux";
      GOARCH = "amd64";
      CC = "${cc}/bin/gcc";
      nativeBuildInputs = [ muslPkgs.pkg-config ];
      buildInputs = [
        cc
        static-pcsclite
      ];
      staticBuild = true;
    };


  # NOTE: building darwin binaries is only supported on macOS.
  # Cross compilation for darwin doesn't work very well with the nix ecosystem.
  # It only works on macOS and even there, cross compilation between architectures
  # is not yet supported. Given these limitations, it makes better sense to use 
  # macOS's clang instead of a C compiler from nix. By doing so, we can build
  # we can build for both  arm64 and amd64.
  darwin =
    let
      darwinShell = arch:
        # Because we're using macOS's own toolchain, it is unnecessary to include nix's
        # PCSC package in the build shell.
        mkCrossBuildShell {
          GOOS = "darwin";
          GOARCH = arch;
          CC = "/usr/bin/clang";
          staticBuild = false;
        };
    in
    {
      aarch64 = darwinShell "arm64";
      x86_64 = darwinShell "amd64";
    };
}

