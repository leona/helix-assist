# Project Rules

## Making code changes ##

When making a code change, follow the phased workflow defined in the `hunk-review` skill. Wait for approval before each next phase. Remove previous step artifacts (`TODO` comments) before starting the next one.

## After Code Changes

Don't relax clippy rules -> #[allow(clippy::*)]
After completing code changes, run validation:
```bash
cargo build && cargo clippy --all-targets && cargo test && cargo fmt
```

Integration tests, run after changes related to api clients

```bash
cargo test -- --include-ignored
```

## Design Rules

1. **Start simple.** Add complexity only when proven needed.
   - Bad: `Vec<(T, Option<usize>)>` for pairing → Good: `counterparty: Option<String>`
   - Bad: `Arc<RefCell<...>>` for cross-references → Good: clone the string

2. **Use existing abstractions.** Don't reimplement parsing.
   - Bad: Manual event iteration + base64 decode in test
   - Good: Use standard `SeiParser.parse_block_signals()`
   - If parser exists, use it. Don't duplicate logic.

3. **Read before editing.** Understand the full context.
   - Bad: Rename fields without checking all usages
   - Good: `grep` for all references first

4. **Types and immutability first.** When data doesn't fit your type, fix it at the boundary (deserializers, `From`/`TryFrom`) — never with manual mutation or pre-processing.
   - Bad: Mutate `serde_json::Value` before deserializing; chain `.strip_prefix("posted ").strip_suffix(" ago")` for every text variant
   - Good: `#[serde(deserialize_with)]`; single regex that captures the pattern regardless of surrounding text
   - Rule: Reaching for `mut` or chained string transforms to shape data = missed abstraction

5. **Invalid states unrepresentable.** If runtime checks (`bail!`, `if` guards) validate argument combinations, the type system wasn't used.
   - Bad: Flat `--sort` flag with `--platform`, runtime `bail!("--sort viewed only for upwork")`
   - Good: Clap subcommands — `list upwork --sort viewed`, `list nofluff` — platform-specific args live only where valid
   - Same for `let-else` / `unreachable!` in caller: push the invariant into the data model (`Job::upwork()` method) so misuse panics at compile time or in one central place

## Documentation

- Check `target/md_docs/` when using unfamiliar APIs, 3rd party crates, or trait/method signature errors
- When writing code, don't add redundant comments. Only minimum if at all
- Write idiomatic Rust
