{ crossSystem ? null }:

import (builtins.fetchTarball {
    # Snapshot from https://github.com/nixos/nixpkgs/tree/nixos-23.05 on Sep 7
    url = "https://github.com/nixos/nixpkgs/archive/4077a0e4ac3356222bc1f0a070af7939c3098535.tar.gz";
    sha256 = "1rvcqq166z72p4c0m5hcgqlbj8hkdzmgva1j0w42wq3q14lvrvfp";
  }) {

  crossSystem = crossSystem;

  overlays = [ (self: super: {
  })];

}
