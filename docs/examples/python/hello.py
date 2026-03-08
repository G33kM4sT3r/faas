def handler(request):
    """Receives the parsed JSON body as a dict, returns a dict."""
    name = request.get("name", "world")
    return {"message": f"Hello, {name}!"}
