{
  description = "Arto - remote approval for Claude Code via ntfy push notifications";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  nixConfig = {
    extra-substituters = [ "https://oschrenk.cachix.org" ];
    extra-trusted-public-keys = [
      "oschrenk.cachix.org-1:3JOMfkq2vFiLw4UsCVwzu8kWFBkuS/3DD5AojcO9pks="
    ];
  };

  outputs =
    { self, nixpkgs }:
    let
      date = builtins.substring 0 8 self.lastModifiedDate;
      version = "0-unstable-${date}";
      commit = self.shortRev or self.dirtyShortRev or "unknown";

      systems = [
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: rec {
        arto = pkgs.buildGoModule {
          pname = "arto";
          inherit version;
          src = self;

          # After go.mod changes: set to lib.fakeHash, `nix build`, paste the hash.
          vendorHash = "sha256-ri7y+7hi24019X+cT/41LladBvbFdqHscMZv8yPuyj8=";

          subPackages = [ "cmd/arto" ];

          ldflags = [
            "-s"
            "-w"
            "-X github.com/oschrenk/arto/internal/cli.Version=${version}"
            "-X github.com/oschrenk/arto/internal/cli.Commit=${commit}"
            "-X github.com/oschrenk/arto/internal/cli.BuildDate=${date}"
          ];

          nativeBuildInputs = [ pkgs.installShellFiles ];
          postInstall = ''
            installShellCompletion --cmd arto \
              --bash <($out/bin/arto completion bash) \
              --zsh <($out/bin/arto completion zsh) \
              --fish <($out/bin/arto completion fish)
          '';

          meta = {
            description = "Remote approval for Claude Code via ntfy push notifications";
            homepage = "https://github.com/oschrenk/arto";
            mainProgram = "arto";
            platforms = nixpkgs.lib.platforms.darwin;
          };
        };
        default = arto;
      });

      apps = forAllSystems (pkgs: rec {
        arto = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.arto}/bin/arto";
        };
        default = arto;
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go # go, language
            go-task # task runner, taskfile.yml
            go-tools # go, staticcheck
            gopls # go, lsp
          ];
        };
      });
    };
}
