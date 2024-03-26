{ nixpkgs, system }:
targetSystem:
with (import nixpkgs {
  system = system;
  crossSystem.config = targetSystem;
});
pkgs.buildGoModule ({
  pname = "dividat-driver";
  version = "0.3.4";
  vendorHash = "sha256-S8qJorySFPERNp4i9ckeZCPLyx0d/QweS4hm4ghJT4k=";
  nativeBuildInputs = lib.optionals stdenv.targetPlatform.isLinux [ pkgconfig ];
  buildInputs = lib.optionals stdenv.targetPlatform.isWindows [ windows.mingw_w64_pthreads ]
    ++ lib.optionals stdenv.targetPlatform.isLinux [ pcsclite ]
    ++ lib.optionals stdenv.targetPlatform.isDarwin [ darwin.apple_sdk.frameworks.PCSC ];

  src = ../.;
  meta.platforms =
    [
      "x86_64-linux"
      "x86_64-windows"
      "aarch64-darwin"
    ];
})

