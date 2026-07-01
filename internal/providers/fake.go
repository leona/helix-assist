package providers

// TODO: Add FakeProvider for deterministic, quota-free tests.
// It returns fixed JSON responses matching the same rigid schemas as CustomProvider:
//   - completion: { "completions": ["...", "..."] }
//   - chat: { "result": "..." }
// Enable with --handler fake. Use it after validating once against the real custom CLI
// to avoid consuming LLM quota during repeated test runs.
