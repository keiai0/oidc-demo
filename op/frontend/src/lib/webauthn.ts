/**
 * WebAuthn ユーティリティ関数
 * navigator.credentials API とサーバー間の型変換を行う
 */

/** ArrayBuffer を base64url 文字列に変換する */
export function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/** base64url 文字列を ArrayBuffer に変換する */
export function base64urlToBuffer(base64url: string): ArrayBuffer {
  const base64 = base64url.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64 + "=".repeat((4 - (base64.length % 4)) % 4);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

/**
 * サーバーの CredentialCreationOptions レスポンスを
 * navigator.credentials.create() に渡せる形式に変換する
 */
export function prepareCreationOptions(
  serverOptions: Record<string, unknown>
): CredentialCreationOptions {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const publicKey = (serverOptions as any).publicKey;

  // challenge: base64url → ArrayBuffer
  publicKey.challenge = base64urlToBuffer(publicKey.challenge);

  // user.id: base64url → ArrayBuffer
  if (publicKey.user?.id) {
    publicKey.user.id = base64urlToBuffer(publicKey.user.id);
  }

  // excludeCredentials[].id: base64url → ArrayBuffer
  if (publicKey.excludeCredentials) {
    for (const cred of publicKey.excludeCredentials) {
      cred.id = base64urlToBuffer(cred.id);
    }
  }

  return { publicKey };
}

/**
 * サーバーの CredentialRequestOptions レスポンスを
 * navigator.credentials.get() に渡せる形式に変換する
 */
export function prepareRequestOptions(
  serverOptions: Record<string, unknown>
): CredentialRequestOptions {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const publicKey = (serverOptions as any).publicKey;

  // challenge: base64url → ArrayBuffer
  publicKey.challenge = base64urlToBuffer(publicKey.challenge);

  // allowCredentials[].id: base64url → ArrayBuffer
  if (publicKey.allowCredentials) {
    for (const cred of publicKey.allowCredentials) {
      cred.id = base64urlToBuffer(cred.id);
    }
  }

  return { publicKey };
}

/**
 * navigator.credentials.create() の結果をサーバー送信用 JSON に変換する
 */
export function registrationCredentialToJSON(
  credential: PublicKeyCredential
): Record<string, unknown> {
  const response = credential.response as AuthenticatorAttestationResponse;
  return {
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      attestationObject: bufferToBase64url(response.attestationObject),
    },
  };
}

/**
 * navigator.credentials.get() の結果をサーバー送信用 JSON に変換する
 */
export function authenticationCredentialToJSON(
  credential: PublicKeyCredential
): Record<string, unknown> {
  const response = credential.response as AuthenticatorAssertionResponse;
  return {
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      clientDataJSON: bufferToBase64url(response.clientDataJSON),
      authenticatorData: bufferToBase64url(response.authenticatorData),
      signature: bufferToBase64url(response.signature),
      userHandle: response.userHandle
        ? bufferToBase64url(response.userHandle)
        : null,
    },
  };
}
