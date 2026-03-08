const slugify = require("slugify");

function handler(body) {
    const name = body.name || "world";
    return { message: `Hello, ${name}!`, slug: slugify(name, { lower: true }) };
}
