{
  description = "A toolbox for infrastructure management";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "riscv64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      package = pkgs: pkgs.buildGoModule {
        pname = "james";
        version = "0-unstable";
        src = self;
        vendorHash = "sha256-jQ7YGjNjgVDD7h0OCBWQga2HdS1peF+4bZaIkq8R1w0=";
        subPackages = [ "cmd/james" ];
        env.CGO_ENABLED = 0;

        meta = {
          description = "A toolbox for infrastructure management";
          homepage = "https://github.com/function61/james";
          license = pkgs.lib.licenses.asl20;
          mainProgram = "james";
          platforms = pkgs.lib.platforms.linux;
        };
      };
    in
    {
      overlays.default = final: prev: {
        james = package final;
      };

      packages = forAllSystems (system: {
        default = package nixpkgs.legacyPackages.${system};
      });
    };
}
