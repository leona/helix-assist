{
  description = "helix-assist LSP server";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = {
    self,
    nixpkgs,
  }: let
    systems = ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"];
    forEach = f: nixpkgs.lib.genAttrs systems f;
    version = self.rev or "dev";
  in {
    packages = forEach (system: let
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      default = pkgs.buildGoModule {
        pname = "helix-assist";
        inherit version;
        src = ./.;
        vendorHash = null;
        ldflags = ["-X main.Version=${version}"];
        meta.mainProgram = "helix-assist";
      };
    });
  };
}
