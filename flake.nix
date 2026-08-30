{
  description = "Texlr — polished LaTeX handoffs for coding agents";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-darwin" "x86_64-darwin" "aarch64-linux" "x86_64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
          browserTools = pkgs.lib.optionals pkgs.stdenv.hostPlatform.isLinux [ pkgs.chromium ];
          runtimeTools = (with pkgs; [ tectonic graphviz mermaid-cli ghostscript ]) ++ browserTools;
        in
        {
          default = pkgs.buildGoModule {
            pname = "texlr";
            version = "0.1.0";
            src = self;
            vendorHash = null;
            nativeBuildInputs = [ pkgs.makeWrapper ];
            ldflags = [ "-s" "-w" "-X main.version=0.1.0" ];
            postInstall = ''
              wrapProgram $out/bin/texlr \
                --prefix PATH : ${pkgs.lib.makeBinPath runtimeTools}
            '';
            meta = {
              description = "Build polished, self-contained LaTeX handoff documents";
              homepage = "https://github.com/drdreo/texlr";
              license = [ pkgs.lib.licenses.mit pkgs.lib.licenses.cc-by-40 ];
              mainProgram = "texlr";
            };
          };
        });

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/texlr";
        };
      });

      devShells = forAllSystems (system:
        let pkgs = import nixpkgs { inherit system; };
        in {
          default = pkgs.mkShell {
            packages = (with pkgs; [ go gopls gotools tectonic graphviz mermaid-cli ghostscript ])
              ++ pkgs.lib.optionals pkgs.stdenv.hostPlatform.isLinux [ pkgs.chromium ];
          };
        });
    };
}
