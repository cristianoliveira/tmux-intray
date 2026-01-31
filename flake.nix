{
  description = "Tmux Intray - a tmux notification manager";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    utils.url = "github:numtide/flake-utils";
  };
  outputs = { nixpkgs, utils, ... }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            bash
            bats
            shellcheck
          ];
        };

        packages = {
          default = pkgs.stdenv.mkDerivation {
            pname = "tmux-intray";
            version = "0.1.0";
            src = ./.;
            installPhase = ''
              mkdir -p $out/bin
              cp bin/tmux-intray $out/bin/tmux-intray
              chmod +x $out/bin/tmux-intray
            '';
          };
        };
    });
}
