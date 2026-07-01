use serde_json::{Value, json};
use std::io::{BufRead, BufReader, Read, Write};
use std::process::{Child, ChildStdin, ChildStdout, Command, Stdio};
use std::sync::mpsc;
use std::thread;
use std::time::{Duration, Instant};
use tempfile::TempDir;

#[derive(Debug, Clone, serde::Deserialize, serde::Serialize)]
struct JsonRpcMessage {
    jsonrpc: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    id: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    method: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    params: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    result: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<Value>,
}

struct LspClient {
    child: Child,
    stdin: ChildStdin,
    rx: mpsc::Receiver<JsonRpcMessage>,
    next_id: i64,
    _dir: TempDir,
    uri: String,
}

const TIMEOUT: Duration = Duration::from_secs(30);

impl LspClient {
    fn spawn(handler_args: &[&str]) -> std::io::Result<Self> {
        let dir = TempDir::new()?;
        let path = dir.path().join("test.rs");
        let uri = format!("file://{}", path.to_string_lossy());
        let log_path = dir.path().join("helix-assist.log");

        let mut args: Vec<String> = handler_args.iter().map(ToString::to_string).collect();
        args.push("--log-file".to_string());
        args.push(log_path.to_string_lossy().to_string());

        let mut child = Command::new("./build/helix-assist")
            .args(&args)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()?;
        let stdin = child.stdin.take().expect("child stdin");
        let stdout = child.stdout.take().expect("child stdout");
        let (tx, rx) = mpsc::channel();
        thread::spawn(move || {
            let mut reader = BufReader::new(stdout);
            while let Some(msg) = read_message(&mut reader) {
                if tx.send(msg).is_err() {
                    break;
                }
            }
        });
        Ok(LspClient {
            child,
            stdin,
            rx,
            next_id: 1,
            _dir: dir,
            uri,
        })
    }

    fn uri(&self) -> &str {
        &self.uri
    }

    fn request(&mut self, method: &str, params: &Value) -> Option<JsonRpcMessage> {
        let id = self.next_id;
        self.next_id += 1;
        let msg = json!({
            "jsonrpc": "2.0",
            "id": id,
            "method": method,
            "params": params,
        });
        self.send(&msg);
        self.wait_for(id)
    }

    fn notify(&mut self, method: &str, params: &Value) {
        let msg = json!({
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
        });
        self.send(&msg);
    }

    fn send(&mut self, msg: &Value) {
        let body = serde_json::to_string(msg).expect("serialize json-rpc message");
        let header = format!("Content-Length: {}\r\n\r\n{}", body.len(), body);
        self.stdin
            .write_all(header.as_bytes())
            .expect("write header");
        self.stdin.flush().expect("flush stdin");
    }

    fn wait_for(&mut self, id: i64) -> Option<JsonRpcMessage> {
        let deadline = Instant::now() + TIMEOUT;
        while Instant::now() < deadline {
            match self.rx.recv_timeout(Duration::from_millis(50)) {
                Ok(msg) if msg.id == Some(id) => return Some(msg),
                Ok(_) | Err(mpsc::RecvTimeoutError::Timeout) => {}
                Err(mpsc::RecvTimeoutError::Disconnected) => return None,
            }
        }
        None
    }

    fn shutdown(&mut self) {
        let _ = self.request("shutdown", &Value::Null);
        self.notify("exit", &Value::Null);
        let _ = self.child.wait();
    }
}

impl Drop for LspClient {
    fn drop(&mut self) {
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

fn read_message(reader: &mut BufReader<ChildStdout>) -> Option<JsonRpcMessage> {
    let mut length = None;
    loop {
        let mut line = String::new();
        if reader.read_line(&mut line).ok()? == 0 {
            return None;
        }
        let line = line.trim();
        if line.is_empty() {
            break;
        }
        if let Some(rest) = line.strip_prefix("Content-Length:") {
            length = rest.trim().parse().ok();
        }
    }
    let length = length?;
    let mut buf = vec![0u8; length];
    reader.read_exact(&mut buf).ok()?;
    serde_json::from_slice(&buf).ok()
}

fn init(client: &mut LspClient) {
    let resp = client
        .request(
            "initialize",
            &json!({
                "processId": 1_i64,
                "rootUri": "file:///tmp/test",
                "capabilities": {}
            }),
        )
        .expect("initialize response");
    assert!(resp.result.is_some(), "initialize returned no result");
    let result = resp.result.expect("initialize result");
    assert!(result.get("capabilities").is_some(), "capabilities missing");
    client.notify("initialized", &json!({}));
}

fn did_open(client: &mut LspClient) {
    client.notify(
        "textDocument/didOpen",
        &json!({
            "textDocument": {
                "uri": client.uri(),
                "languageId": "rust",
                "version": 1,
                "text": "fn main() {\n\n}\n"
            }
        }),
    );
}

fn did_change(client: &mut LspClient) {
    client.notify(
        "textDocument/didChange",
        &json!({
            "textDocument": {"uri": client.uri(), "version": 2},
            "contentChanges": [{"text": "fn main() {\n\n}\n"}]
        }),
    );
}

#[test]
#[allow(clippy::mutable_key_type)]
fn test_fake_flow() {
    let mut client = LspClient::spawn(&["--handler", "fake"]).expect("spawn helix-assist");
    init(&mut client);
    did_open(&mut client);
    did_change(&mut client);

    let comp = client
        .request(
            "textDocument/completion",
            &json!({
                "textDocument": {"uri": client.uri()},
                "position": {"line": 0, "character": 11}
            }),
        )
        .expect("completion response");
    let result = comp.result.expect("completion result");
    let list: lsp_types::CompletionList = serde_json::from_value(result).expect("completion list");
    assert!(!list.items.is_empty(), "expected completion items");

    let action = client
        .request(
            "textDocument/codeAction",
            &json!({
                "textDocument": {"uri": client.uri()},
                "range": {
                    "start": {"line": 0, "character": 0},
                    "end": {"line": 1, "character": 0}
                },
                "context": {"diagnostics": []}
            }),
        )
        .expect("codeAction response");
    let actions: Vec<lsp_types::CodeAction> =
        serde_json::from_value(action.result.expect("codeAction result")).expect("codeAction list");
    assert_eq!(actions.len(), 3, "expected three code actions");

    let exec = client
        .request(
            "workspace/executeCommand",
            &json!({
                "command": "improveCode",
                "arguments": [{
                    "range": {
                        "start": {"line": 0, "character": 0},
                        "end": {"line": 1, "character": 1}
                    },
                    "query": "Improve this code",
                    "diagnostics": []
                }]
            }),
        )
        .expect("executeCommand response");
    assert_eq!(
        exec.method.as_deref(),
        Some("workspace/applyEdit"),
        "expected workspace/applyEdit request"
    );
    let params = exec.params.expect("applyEdit params");
    let edit: lsp_types::ApplyWorkspaceEditParams =
        serde_json::from_value(params).expect("workspace edit");
    let changes = edit.edit.changes.expect("workspace edit changes");
    assert!(
        !changes.is_empty(),
        "expected non-empty workspace edit changes"
    );

    let shutdown = client
        .request("shutdown", &Value::Null)
        .expect("shutdown response");
    assert!(
        shutdown.result.is_none(),
        "expected null result for shutdown"
    );

    client.shutdown();
}

#[test]
#[ignore = "requires live pi LLM and API key"]
#[allow(clippy::mutable_key_type)]
fn test_pi_live_flow() {
    let mut client = LspClient::spawn(&[
        "--handler",
        "pi --print --no-session --no-tools --no-extensions --mode text --thinking off --model deepseek/deepseek-v4-flash",
    ])
    .expect("spawn helix-assist with pi handler");
    init(&mut client);
    did_open(&mut client);
    did_change(&mut client);

    let comp = client
        .request(
            "textDocument/completion",
            &json!({
                "textDocument": {"uri": client.uri()},
                "position": {"line": 0, "character": 11}
            }),
        )
        .expect("completion response");
    let result = comp.result.expect("completion result");
    let list: lsp_types::CompletionList = serde_json::from_value(result).expect("completion list");
    assert!(!list.items.is_empty(), "expected completion items from pi");

    let _ = client
        .request(
            "textDocument/codeAction",
            &json!({
                "textDocument": {"uri": client.uri()},
                "range": {
                    "start": {"line": 0, "character": 0},
                    "end": {"line": 1, "character": 0}
                },
                "context": {"diagnostics": []}
            }),
        )
        .expect("codeAction response");

    let exec = client
        .request(
            "workspace/executeCommand",
            &json!({
                "command": "improveCode",
                "arguments": [{
                    "range": {
                        "start": {"line": 0, "character": 0},
                        "end": {"line": 1, "character": 1}
                    },
                    "query": "Improve this code",
                    "diagnostics": []
                }]
            }),
        )
        .expect("executeCommand response");
    assert_eq!(
        exec.method.as_deref(),
        Some("workspace/applyEdit"),
        "expected workspace/applyEdit from pi chat"
    );
    let params = exec.params.expect("applyEdit params");
    let edit: lsp_types::ApplyWorkspaceEditParams =
        serde_json::from_value(params).expect("workspace edit");
    let changes = edit.edit.changes.expect("workspace edit changes");
    assert!(
        !changes.is_empty(),
        "expected non-empty workspace edit changes"
    );

    client.shutdown();
}
