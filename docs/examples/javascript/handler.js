/**
 * Receives the parsed JSON body as an object, returns an object
 * that will be serialized as the JSON response.
 */
function handler(body) {
    const name = body.name || "world";
    return { message: `Hello, ${name}!` };
}
