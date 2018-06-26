# This pins the version of nixpkgs
let
  _nixpkgs = import <nixpkgs> {};
in 
  import (_nixpkgs.fetchFromGitHub 
  { owner = "NixOS"
  ; repo = "nixpkgs"
  ; rev = "9af0ed346d710afbaa37b44b651199381c37543a"
  ; sha256 = "1n37piclgaqp8nsk439rhawqpdchj9fbvx8qbxxrhmh3bh7mwxyh"; })

