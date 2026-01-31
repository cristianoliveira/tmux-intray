{
  description = "Tmux Intray - a tmux notification manager";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    utils.url = "github:numtide/flake-utils";
    git-hooks.url = "github:cachix/git-hooks.nix";
  };
  outputs = { nixpkgs, utils, git-hooks, ... }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        pre-commit-check = git-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            shellcheck = {
              enable = true;
              files = "\\.(sh|bats)$";
              exclude = "^tests/";
              args = [ "-e" "SC2034" "-e" "SC1091" ];
            };
            shfmt = {
              enable = true;
              files = "\\.(sh|bats|tmux)$";
              # auto-fix enabled
              entry = "${pkgs.shfmt}/bin/shfmt -w -i 4";
            };
            bats = {
              enable = true;
              files = "^tests/.*\\.bats$";
              # run bats on modified test files
              entry = "${pkgs.bats}/bin/bats";
            };
          };
        };
      in {
        checks = {
          pre-commit-check = pre-commit-check;
        };

        devShells.default = pkgs.mkShell {
          inherit (pre-commit-check) shellHook;
          packages = with pkgs; [
            bash
            bats
            shellcheck
            shfmt

            pre-commit
          ];
          buildInputs = pre-commit-check.enabledPackages;
        };

        packages = {
          default = pkgs.stdenv.mkDerivation {
            pname = "tmux-intray";
            version = "0.1.0";
            src = ./.;
            nativeBuildInputs = with pkgs; [ bats tmux shfmt ];
            preBuild = ''
              export XDG_STATE_HOME=$(mktemp -d)
              export XDG_CONFIG_HOME=$(mktemp -d)
              export HOME=$(mktemp -d)
            '';
            installPhase = ''
              mkdir -p $out/bin
              cp bin/tmux-intray $out/bin/tmux-intray
              chmod +x $out/bin/tmux-intray
            '';
          };
        };
    });
}