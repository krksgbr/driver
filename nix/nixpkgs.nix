{ crossSystem ? null }:

import ((import <nixpkgs> {}).fetchFromGitHub {
  owner = "NixOS";
  repo = "nixpkgs";
  rev = "18.09";
  sha256 = "1ib96has10v5nr6bzf7v8kw7yzww8zanxgw2qi1ll1sbv6kj6zpd";
}) {

  crossSystem = crossSystem;

  overlays = [ (self: super: {

    # Backport Go 1.12.9 to old Nixpkgs
    # Nixpkgs 19.03 and up seem to break crossbuilding with musl
    # Based on https://github.com/NixOS/nixpkgs/blob/19.09/pkgs/development/compilers/go/1.12.nix
    go_1_12 = super.go_1_11.overrideAttrs (old: rec {
      version = "1.12.9";
      name = "go-${version}";
      src = self.fetchFromGitHub {
        owner = "golang";
        repo = "go";
        rev = "go${version}";
        sha256 = "1q316wgxhskwn5p622bcv81dhg93mads1591fppcf0dwyzpnl6wb";
      };
      patches = (
        (self.lib.filter
          (x: !(self.lib.hasSuffix "ssl-cert-file-1.9.patch" (builtins.toString x)) && !(self.lib.hasSuffix "remove-fhs-test-references.patch" (builtins.toString x)))
          old.patches
        ) ++ [
          (self.fetchurl {
            url = "https://github.com/NixOS/nixpkgs/raw/19.09/pkgs/development/compilers/go/ssl-cert-file-1.12.1.patch";
            sha256 = "1645yrz36w35lnpalin4ygg39s7hpllamf81w1yr08g8div227f1";
          })
        ]
      );
      GOCACHE = null;
      GO_BUILDER_NAME = "nix";
      configurePhase = "";
      postConfigure = ''
        export GOCACHE=$TMPDIR/go-cache
        # this is compiled into the binary
        export GOROOT_FINAL=$out/share/go
        export PATH=$(pwd)/bin:$PATH
        # Independent from host/target, CC should produce code for the building system.
        export CC=${self.buildPackages.stdenv.cc}/bin/cc
        ulimit -a
      '';
      postBuild = ''
        (cd src && ./make.bash)
      '';
      preInstall = ''
        #rm -r pkg/{bootstrap,obj}
        # Contains the wrong perl shebang when cross compiling,
        # since it is not used for anything we can deleted as well.
        rm src/regexp/syntax/make_perl_groups.pl
      '' + (if (self.stdenv.buildPlatform != self.stdenv.hostPlatform) then ''
        mv bin/*_*/* bin
        rmdir bin/*_*
        ${self.optionalString (!(self.GOHOSTARCH == self.GOARCH && self.GOOS == self.GOHOSTOS)) ''
          rm -rf pkg/${self.GOHOSTOS}_${self.GOHOSTARCH} pkg/tool/${self.GOHOSTOS}_${self.GOHOSTARCH}
        ''}
      '' else if (self.stdenv.hostPlatform != self.stdenv.targetPlatform) then ''
        rm -rf bin/*_*
        ${self.optionalString (!(self.GOHOSTARCH == self.GOARCH && self.GOOS == self.GOHOSTOS)) ''
          rm -rf pkg/${self.GOOS}_${self.GOARCH} pkg/tool/${self.GOOS}_${self.GOARCH}
        ''}
      '' else "");
      installPhase = ''
        runHook preInstall
        mkdir -p $GOROOT_FINAL
        cp -a bin pkg src lib misc api doc $GOROOT_FINAL
        ln -s $GOROOT_FINAL/bin $out/bin
        runHook postInstall
      '';
    });

    # GOCACHE can not be disabled in Go 1.12 but buildGoPackage definition hardcodes
    # turning it off in NixOS 18.09.
    buildGo112Package =
      self.callPackage (self.fetchurl { url = "https://github.com/NixOS/nixpkgs/raw/19.09/pkgs/development/go-packages/generic/default.nix"; sha256 = "1bwkjbfxfym3v6z2zv0yrygpzck2cx63dpv46jil3py0yndaqrwa"; }) {
        go = self.go_1_12;
      };

  })];

}
