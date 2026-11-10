{
  description = "Golang shell with btrfs-progs";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  inputs.systems.url = "github:nix-systems/default";
  inputs.flake-utils = {
    url = "github:numtide/flake-utils";
    inputs.systems.follows = "systems";
  };

  outputs = {
    nixpkgs,
    flake-utils,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
        sudoKeepEnv = pkgs.writeShellScriptBin "sudo" ''
          exec /run/wrappers/bin/sudo -E ${pkgs.coreutils}/bin/env "PATH=$PATH" "$@"
        '';
      in {
        devShells.default = pkgs.mkShell {
          nativeBuildInputs = with pkgs; [pkg-config go_1_27 gopls delve gotools sudoKeepEnv];
          buildInputs = with pkgs; [btrfs-progs];
          packages = with pkgs; [util-linux];
        };
      }
    );
}
