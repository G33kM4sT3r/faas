<?php
function handler(array $input): array {
    $name = $input['name'] ?? 'world';
    return ['message' => "Hello, {$name}!"];
}
