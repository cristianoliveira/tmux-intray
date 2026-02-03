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
            shfmt

            prek

            ## For tmux
            go

            golangci-lint

            # Test runner with good output
            # USAGE: gotestsum --watch
            gotestsum

            # To create new subcommands, run:
            # cobra-cli add <subcommand-name>
            cobra-cli

            # To generate the mock for the interfaces, run:
            # mockgen -source=./pkg/cli/cli.go -destination=./pkg/cli/mock/mock_cli.go -package=mock
            mockgen
          ];
        };

        packages = {
          default = pkgs.stdenv.mkDerivation {
            pname = "tmux-intray";
            version = "0.1.0";
            src = ./.;
            nativeBuildInputs = with pkgs; [ bats tmux shfmt shellcheck ];
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
