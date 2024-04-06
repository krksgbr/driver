{ nixpkgs, system, version }:
let
  # ld flag to embed the version in the binary
  ldEmbedVersion =
    "-X github.com/dividat/driver/src/dividat-driver/server.version=${version}";

  buildGoModuleFor = targetSystem: customAttrs:
    let
      crossPkgs = (import nixpkgs {
        system = system;
        crossSystem.config = targetSystem;
      });
    in
    with crossPkgs;
    buildGoModule ({
      pname = "dividat-driver";
      version = version;
      vendorHash = "sha256-S8qJorySFPERNp4i9ckeZCPLyx0d/QweS4hm4ghJT4k=";
      ldflags = [
        ldEmbedVersion
        "-linkmode external"
        "-extldflags '-static'"
      ];
      src = ../.;
      meta.platforms =
        [
          "x86_64-linux"
          "x86_64-windows"
        ];
    } // customAttrs crossPkgs);
in
{
  x86_64-windows = buildGoModuleFor "x86_64-w64-mingw32" (pkgs: {
    buildInputs = [ pkgs.windows.mingw_w64_pthreads ];
  });

  x86_64-linux =
    buildGoModuleFor "x86_64-unknown-linux-musl" (pkgs: {
      nativeBuildInputs = [ pkgs.pkg-config ];
      buildInputs =
        let
          static-pcsclite = pkgs.pcsclite.overrideAttrs (attrs: {
            configureFlags = attrs.configureFlags ++ [ "--enable-static" "--disable-shared" ];
          });
        in
        [ static-pcsclite ];
    });

  # Cross compilation for darwin doesn't work very well with the nix ecosystem.
  # Only MacOS is supported as a build platform and even there, cross compilation
  # between architectures is not yet supported.
  # We can leverage go's built-in cross compilation capabilities, and use the system's
  # clang compiler shipped with MacOS. This setup is simple and it works smoothly, but
  # obviously it will only work on MacOS. 
  darwinCrossBuildScript =
    let
      buildCmd = arch:
        ''
          OUT="bin/dividat-driver-${arch}"
          GOARCH=${arch} go build -ldflags "${ldEmbedVersion}" -o "$OUT" src/dividat-driver/main.go
          echo "Built $OUT"
        '';
    in
    with (import nixpkgs { inherit system; });
    (writeShellApplication {
      name = "build-for-darwin";
      runtimeInputs = lib.optionals stdenv.isDarwin [ pkgs.go ];
      text =
        if stdenv.isDarwin then ''
          export GOOS=darwin
          export CGO_ENABLED=1
          export CC="/usr/bin/clang"

          mkdir -p bin

          ${buildCmd "amd64"}
          ${buildCmd "arm64"}

        '' else ''
          echo Building darwin binaries is only supported on MacOS.
          exit 1
        '';
    });
}

