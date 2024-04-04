{ crossSystem ? null }:

import ((import <nixpkgs> {}).fetchFromGitHub {
  owner = "NixOS";
  repo = "nixpkgs";
  rev = "4077a0e4ac3356222bc1f0a070af7939c3098535"; # 23.05
  sha256 = "sha256-1+28KQl4YC4IBzKo/epvEyK5KH4MlgoYueJ8YwLGbOc=";
}) {
  crossSystem = crossSystem;
}
