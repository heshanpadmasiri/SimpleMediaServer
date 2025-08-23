{
  description = "SimpleMediaServer Go development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            fish
          ];

          shellHook = ''
            export CGO_CFLAGS="-U_FORTIFY_SOURCE"
            export CGO_CFLAGS_ALLOW="-D_FORTIFY_SOURCE.*"
            export SHELL=${pkgs.fish}/bin/fish
            exec ${pkgs.fish}/bin/fish
          '';
        };
      });
}
