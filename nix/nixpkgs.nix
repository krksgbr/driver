# This pins the version of nixpkgs
let
  _nixpkgs = import <nixpkgs> {};
in 
  import (_nixpkgs.fetchFromGitHub 
  { owner = "NixOS"
  ; repo = "nixpkgs"
  ; rev = "18.09"
  ; sha256 = "1ib96has10v5nr6bzf7v8kw7yzww8zanxgw2qi1ll1sbv6kj6zpd"; })

