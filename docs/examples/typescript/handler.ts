/**
 * Receives the parsed JSON body as a record, returns an object
 * that will be serialized as the JSON response.
 */
function handler(body: Record<string, any>): Record<string, any> {
    const name = body.name || "world";
    return { message: `Hello, ${name}!` };
}
