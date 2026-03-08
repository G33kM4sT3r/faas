use slug::slugify;

fn handler(input: Value) -> Value {
    let name = input.get("name")
        .and_then(|v| v.as_str())
        .unwrap_or("world");
    json!({"message": format!("Hello, {}!", name), "slug": slugify(name)})
}
