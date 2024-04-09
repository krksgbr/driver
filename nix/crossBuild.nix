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
    let
      staticLinkingLdFlags =
        if staticBuild then
          "-linkmode external -extldflags '-static'"
        else
          "";

      buildScript =
        (pkgs.writeShellApplication {
          name = "build-driver";
          text =
            ''
              IN=""
              OUT=""
              VERSION=""
      
              while getopts "i:o:v:" opt; do
                case $opt in
                  i) IN="$OPTARG"
                     ;;
                  o) OUT="$OPTARG"
                     ;;
                  v) VERSION="$OPTARG"
                     ;;
                  \?) echo "Invalid option: -$OPTARG" >&2; exit 1 ;;
                esac
              done
      
              all_flags_set=true
              ensure_flag_set() {
                local flag_name="$1"
                local flag_value="$2"
                if [ -z "$flag_value" ]; then
                  echo "$flag_name not set"
                  all_flags_set=false
                fi
              }
      
              ensure_flag_set "-v" "$VERSION"
              ensure_flag_set "-i" "$IN"
              ensure_flag_set "-o" "$OUT"
      
              if [ "$all_flags_set" = false ]; then
                echo "Usage: build-driver -v <version> -i <input> -o <output>"
                exit 1
              fi

              LD_VERSION="-X github.com/dividat/driver/src/dividat-driver/server.version=$VERSION"
              STATIC_LINKING_LDFLAGS="${staticLinkingLdFlags}"

              export GOOS=${GOOS}
              export GOARCH=${GOARCH}
              export CGO_ENABLED="1"
              export CC=${CC}

              go build  -ldflags "$LD_VERSION $STATIC_LINKING_LDFLAGS" -o "$OUT" "$IN"
            '';
        });
    in
    pkgs.mkShell {
      inherit nativeBuildInputs;
      buildInputs = [
        pkgs.go
        buildScript
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

