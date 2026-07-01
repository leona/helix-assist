package providers

// TODO: Add CustomProvider that calls a configurable CLI as a subprocess.
// The CLI must return JSON matching a rigid schema for each LSP method:
//   - completion: { "completions": ["...", "..."] }
//   - chat: { "result": "..." }
// Implementation in Go:
//   1. Define typed structs with `encoding/json` tags.
//   2. Build the prompt by embedding the expected JSON schema as text.
//   3. Run the CLI with the prompt via os/exec.
//   4. Parse stdout with json.Unmarshal into the typed struct.
//   5. On unmarshal error, return an error so the LSP handler surfaces it.
// No external JSON schema libraries needed; keep it simple and manual.