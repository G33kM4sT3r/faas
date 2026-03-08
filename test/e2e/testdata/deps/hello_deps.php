<?php
use Cocur\Slugify\Slugify;

function handler(array $input): array {
    $name = $input['name'] ?? 'world';
    $slugify = new Slugify();
    return ['message' => "Hello, {$name}!", 'slug' => $slugify->slugify($name)];
}
