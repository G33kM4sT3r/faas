function handler(body) {
    const name = body.name || "world";
    return { message: `Hello, ${name}!` };
}
