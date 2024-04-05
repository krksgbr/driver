{ nixpkgs, system, version }: targetSystem:
let
  pkgs = (import nixpkgs {
    system = system;
    crossSystem.config = targetSystem;
  });
  static-pcsclite = pkgs.pcsclite.overrideAttrs (attrs: {
    configureFlags = attrs.configureFlags ++ [ "--enable-static" "--disable-shared" ];
  });
in
with pkgs;
buildGoModule ({
  pname = "dividat-driver";
  version = version;
  vendorHash = "sha256-S8qJorySFPERNp4i9ckeZCPLyx0d/QweS4hm4ghJT4k=";
  ldflags = [
    "-X github.com/dividat/driver/src/dividat-driver/server.version=${version}"
  ] ++ lib.optionals (!stdenv.targetPlatform.isDarwin) [
    "-linkmode external"
    "-extldflags '-static'"
  ];
  nativeBuildInputs = lib.optionals stdenv.targetPlatform.isLinux [ pkg-config ];
  buildInputs = lib.optionals stdenv.targetPlatform.isLinux [ static-pcsclite ]
    ++ lib.optionals stdenv.targetPlatform.isWindows [ windows.mingw_w64_pthreads ]
    ++ lib.optionals stdenv.targetPlatform.isDarwin [ darwin.apple_sdk.frameworks.PCSC ];
  src = ../.;
  meta.platforms =
    [
      "x86_64-linux"
      "x86_64-windows"
      "aarch64-darwin"
    ];
})

