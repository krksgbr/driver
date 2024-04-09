# Shell script to build the driver.
# It's available inside the default development shell,
# and is also used by the cross-build shells (crossBuild.nix).
# It is meant to be called from the Makefile.
#
# Usage: build-driver -v <version> -i <input> -o <output>
# See Makefile for concrete examples.

{ pkgs
, GOOS
, GOARCH
, CC
, staticBuild
}:
let
  staticLinkingLdFlags =
    if staticBuild then
      "-linkmode external -extldflags '-static'"
    else
      "";
in
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
})
