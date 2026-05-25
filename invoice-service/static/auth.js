(function () {
  const usernameMessage = 'Use 3-32 characters: lowercase letters, numbers, dots, underscores, or hyphens. Start with a letter or number.';
  const usernamePattern = /^[a-z0-9][a-z0-9._-]{2,31}$/;

  function base64urlToBuffer(base64url) {
    const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
    const padding = '='.repeat((4 - base64.length % 4) % 4);
    const binary = atob(base64 + padding);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    return bytes.buffer;
  }

  function bufferToBase64url(buf) {
    if (!buf) {
      throw new Error('Expected credential data was missing.');
    }
    const bytes = new Uint8Array(buf);
    let binary = '';
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
  }

  function normalizeUsername(value) {
    return value.trim().toLowerCase();
  }

  function validatedUsername(input) {
    const username = normalizeUsername(input.value);
    input.value = username;
    if (!usernamePattern.test(username)) {
      throw new Error(usernameMessage);
    }
    return username;
  }

  function statusEl() {
    return document.getElementById('auth-status');
  }

  function setStatus(message, kind) {
    const el = statusEl();
    if (!el) return;
    el.textContent = message || '';
    el.className = kind ? 'auth-status auth-status-' + kind : 'auth-status';
  }

  async function jsonOrDefault(response, fallback) {
    try {
      return await response.json();
    } catch (_) {
      return { error: fallback };
    }
  }

  function loginCredentialToJSON(cred) {
    const json = {
      id: cred.id,
      type: cred.type,
      rawId: bufferToBase64url(cred.rawId),
      response: {
        clientDataJSON: bufferToBase64url(cred.response.clientDataJSON),
        authenticatorData: bufferToBase64url(cred.response.authenticatorData),
        signature: bufferToBase64url(cred.response.signature),
      },
    };
    if (cred.response.userHandle) {
      json.response.userHandle = bufferToBase64url(cred.response.userHandle);
    }
    return json;
  }

  function registrationCredentialToJSON(cred) {
    return {
      id: cred.id,
      type: cred.type,
      rawId: bufferToBase64url(cred.rawId),
      response: {
        clientDataJSON: bufferToBase64url(cred.response.clientDataJSON),
        attestationObject: bufferToBase64url(cred.response.attestationObject),
        transports: cred.response.getTransports ? cred.response.getTransports() : [],
      },
    };
  }

  function decodeLoginOptions(options) {
    options.publicKey.challenge = base64urlToBuffer(options.publicKey.challenge);
    if (options.publicKey.allowCredentials) {
      for (const cred of options.publicKey.allowCredentials) {
        cred.id = base64urlToBuffer(cred.id);
      }
    }
    return options;
  }

  function decodeRegistrationOptions(options) {
    options.publicKey.challenge = base64urlToBuffer(options.publicKey.challenge);
    options.publicKey.user.id = base64urlToBuffer(options.publicKey.user.id);
    if (options.publicKey.excludeCredentials) {
      for (const cred of options.publicKey.excludeCredentials) {
        cred.id = base64urlToBuffer(cred.id);
      }
    }
    return options;
  }

  function safeRedirect(next) {
    if (next && next.startsWith('/') && !next.startsWith('//') && !next.includes('\\')) {
      window.location.href = next;
      return;
    }
    window.location.href = '/';
  }

  async function beginLogin(event) {
    event.preventDefault();
    const form = event.currentTarget;
    const input = document.getElementById('username');
    const button = document.getElementById('login-btn');
    const originalText = button.textContent;

    button.disabled = true;
    setStatus('', '');

    try {
      const username = validatedUsername(input);
      button.textContent = 'Signing in...';

      const beginResp = await fetch('/login/begin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
        body: JSON.stringify({ username }),
      });
      if (!beginResp.ok) {
        const err = await jsonOrDefault(beginResp, 'login failed');
        throw new Error(err.message || 'Login failed. Check your username and passkey, then try again.');
      }

      const { session_id, options } = await beginResp.json();
      const credential = await navigator.credentials.get(decodeLoginOptions(options));

      const finishResp = await fetch('/login/finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
        body: JSON.stringify({ session_id, ...loginCredentialToJSON(credential) }),
      });
      if (!finishResp.ok) {
        const err = await jsonOrDefault(finishResp, 'login failed');
        throw new Error(err.message || 'Login failed. Check your username and passkey, then try again.');
      }

      safeRedirect(form.dataset.next || '/');
    } catch (error) {
      setStatus(error.message || 'Login failed. Check your username and passkey, then try again.', 'error');
      button.disabled = false;
      button.textContent = originalText;
    }
  }

  async function beginRegister(event) {
    event.preventDefault();
    const input = document.getElementById('username');
    const button = document.getElementById('register-btn');
    const originalText = button.textContent;

    button.disabled = true;
    setStatus('', '');

    try {
      const username = validatedUsername(input);
      button.textContent = 'Creating...';

      const beginResp = await fetch('/register/begin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
        body: JSON.stringify({ username, displayName: username }),
      });
      if (!beginResp.ok) {
        const err = await jsonOrDefault(beginResp, 'registration failed');
        throw new Error(err.message || 'Registration failed. Try again later.');
      }

      const { session_id, options } = await beginResp.json();
      const credential = await navigator.credentials.create(decodeRegistrationOptions(options));

      const finishResp = await fetch('/register/finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
        body: JSON.stringify({ session_id, ...registrationCredentialToJSON(credential) }),
      });
      if (!finishResp.ok) {
        const err = await jsonOrDefault(finishResp, 'registration failed');
        throw new Error(err.message || 'Registration failed. Try again later.');
      }

      safeRedirect('/');
    } catch (error) {
      setStatus(error.message || 'Registration failed. Try again later.', 'error');
      button.disabled = false;
      button.textContent = originalText;
    }
  }

  const loginForm = document.getElementById('login-form');
  if (loginForm) {
    loginForm.addEventListener('submit', beginLogin);
  }

  const registerForm = document.getElementById('register-form');
  if (registerForm) {
    registerForm.addEventListener('submit', beginRegister);
  }
})();
