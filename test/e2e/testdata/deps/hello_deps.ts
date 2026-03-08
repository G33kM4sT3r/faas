import slugify from "slugify";

function handler(body: unknown): unknown {
    const input = body as Record<string, unknown>;
    const name = (input.name as string) || "world";
    return { message: `Hello, ${name}!`, slug: slugify(name, { lower: true }) };
}
