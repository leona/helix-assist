// TODO: Add Rust LSP protocol integration tests that spawn the helix-assist binary
// and verify JSON-RPC responses for: initialize, textDocument/didOpen,
// textDocument/didChange, textDocument/completion, textDocument/codeAction,
// workspace/executeCommand, shutdown, and exit.
// Use --handler fake for fast, deterministic, quota-free tests.
// Add an #[ignored] test that uses --handler custom with the real pi CLI to validate
// against a live LLM when needed.
