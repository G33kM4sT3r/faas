<?php

/**
 * Receives the parsed JSON body as an associative array,
 * returns an associative array that will be serialized as the JSON response.
 */
function handler(array $input): array
{
    $name = $input['name'] ?? 'world';
    return ['message' => "Hello, {$name}!"];
}
