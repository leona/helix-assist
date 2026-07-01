{
  pkgs,
  lib,
  config,
  inputs,
  ...
}: {
  packages = [pkgs.git];

  languages = {
    go = {
      enable = true;
    };
  };

  scripts = {
    go-build.exec = ''
      set -x
      go build -o build/helix-assist ./cmd/helix-assist
    '';

    go-test.exec = ''
      go test -v ./...
    '';
  };
}
