(() => {
  const root = document.querySelector("#passkeys");
  if (!root) return;
  const status = document.querySelector("#passkey-status");
  const decode = (value) =>
    Uint8Array.from(atob(value.replace(/-/g, "+").replace(/_/g, "/")), (c) =>
      c.charCodeAt(0),
    ).buffer;
  const encode = (value) =>
    btoa(String.fromCharCode(...new Uint8Array(value)))
      .replace(/\+/g, "-")
      .replace(/\//g, "_")
      .replace(/=+$/, "");
  async function request(path, body) {
    const response = await fetch(path, {
      method: "POST",
      headers: {
        "X-CSRF-Token": root.dataset.csrf,
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body || {}),
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "Passkey request failed.");
    return data;
  }
  async function run(register) {
    const button = document.querySelector(
      register ? "#register-passkey" : "#login-passkey",
    );
    button.disabled = true;
    status.textContent = "";
    try {
      if (!window.PublicKeyCredential)
        throw new Error(
          "This browser does not support passkeys. Use another sign-in method.",
        );
      const kind = register ? "register" : "login";
      const data = await request("/passkeys/" + kind + "-begin");
      const options = data.options.publicKey;
      options.challenge = decode(options.challenge);
      if (register) options.user.id = decode(options.user.id);
      for (const key of ["allowCredentials", "excludeCredentials"])
        for (const item of options[key] || []) item.id = decode(item.id);
      const credential = await navigator.credentials[
        register ? "create" : "get"
      ]({ publicKey: options });
      if (!credential) throw new Error("No passkey was selected.");
      const result = {
        id: credential.id,
        rawId: encode(credential.rawId),
        type: credential.type,
        clientExtensionResults: credential.getClientExtensionResults(),
        response: {
          clientDataJSON: encode(credential.response.clientDataJSON),
        },
      };
      if (register) {
        result.response.attestationObject = encode(
          credential.response.attestationObject,
        );
        result.response.transports =
          credential.response.getTransports?.() || [];
      } else {
        result.response.authenticatorData = encode(
          credential.response.authenticatorData,
        );
        result.response.signature = encode(credential.response.signature);
        result.response.userHandle = credential.response.userHandle
          ? encode(credential.response.userHandle)
          : null;
      }
      const query = new URLSearchParams({
        flow: root.dataset.flow || "",
        continue: root.dataset.continue || "",
        name: document.querySelector("#passkey-name")?.value || "",
      });
      const done = await request(
        "/passkeys/" + kind + "-finish?" + query,
        result,
      );
      if (register) location.reload();
      else location.assign(done.next);
    } catch (error) {
      status.textContent =
        error.name === "NotAllowedError"
          ? "Passkey request cancelled or timed out. You can try again."
          : error.message;
    } finally {
      button.disabled = false;
    }
  }
  document
    .querySelector("#register-passkey")
    ?.addEventListener("click", () => run(true));
  document
    .querySelector("#login-passkey")
    ?.addEventListener("click", () => run(false));
})();
