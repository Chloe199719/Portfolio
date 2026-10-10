"""Register the second example application only against the local CI issuer."""

import http.client
import json
import os
from http.cookies import SimpleCookie
from urllib.parse import urlparse

api = urlparse(os.environ.get("API_URL", ""))
if os.environ.get("CI") != "true" or api.scheme != "http" or api.hostname != "localhost":
    raise SystemExit("This helper only runs against the local CI server")

with open(os.environ["E2E_FIXTURE_FILE"]) as f:
    fixture = json.load(f)
connection = http.client.HTTPConnection("localhost", api.port, timeout=10)
connection.request("GET", "/v1/session")
response = connection.getresponse()
assert response.status == 200
cookies = SimpleCookie(response.getheader("Set-Cookie"))
session = json.load(response)
body = json.dumps({
    "name": "CI independent web application",
    "public": False,
    "redirectUris": ["http://localhost:4000/callback"],
    "origins": ["http://localhost:4000"],
    "scopes": ["openid", "profile", "email"],
})
connection.request("POST", "/v1/admin/applications", body, {
    "Content-Type": "application/json",
    "Origin": os.environ["SITE_URL"],
    "X-CSRF-Token": session["csrf"],
    "Cookie": "chloe_csrf=" + cookies["chloe_csrf"].value + "; chloe_api=" + fixture["ownerToken"],
})
response = connection.getresponse()
assert response.status == 201, "Example client registration failed"
print(json.dumps(json.load(response)))
connection.close()
