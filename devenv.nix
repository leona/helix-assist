{
  pkgs,
  lib,
  config,
  inputs,
  ...
}: {
  packages = [pkgs.git];

  languages = {
    rust = {
      enable = true;
      channel = "nightly";
    };
    go = {
      enable = true;
    };
  };

  scripts = {
    test.exec = ''
      cargo build && cargo clippy --all-targets && cargo test && cargo fmt
    '';
    test-integration.exec = ''
      cargo test -- --include-ignored
    '';
    export-docs.exec = ''
      RUSTDOCFLAGS="-Zunstable-options --output-format=json" cargo doc
      cargo docs-md --dir target/doc/ -o target/md_docs  --source-locations --full-method-docs --hide-trivial-derives
    '';
  };
}
