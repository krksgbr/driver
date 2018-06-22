# This pins the version of nixpkgs
let
  _nixpkgs = import <nixpkgs> {};
in 
  import (_nixpkgs.fetchFromGitHub 
  { owner = "NixOS"
  ; repo = "nixpkgs"
  ; rev = "9af0ed346d710afbaa37b44b651199381c37543a"
  ; sha256 = "0cyhrvcgp8hppsvgycr0a0fiz00gcd24vxcxmv22g6dibdf5377h"; })

