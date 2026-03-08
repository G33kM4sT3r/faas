use serde_json::{json, Value};

/// Receives the parsed JSON body as a serde_json::Value,
/// returns a Value that will be serialized as the JSON response.
fn handler(input: Value) -> Value {
    let name = input["name"].as_str().unwrap_or("world");
    json!({"message": format!("Hello, {}!", name)})
}
