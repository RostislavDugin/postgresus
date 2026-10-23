// The OpenID Connect state is a random single-use token rather than a provider
// tag, so the callback page cannot read the provider out of it the way the
// GitHub and Google callbacks do. This marker records which flow the browser
// left on, and is cleared as soon as the callback page reads it.
const oidcSignInMarkerKey = 'oidcSignInStarted';

export function markOidcSignInStarted() {
  sessionStorage.setItem(oidcSignInMarkerKey, 'true');
}

export function readAndClearOidcSignInMarker(): boolean {
  const isStarted = sessionStorage.getItem(oidcSignInMarkerKey) === 'true';
  sessionStorage.removeItem(oidcSignInMarkerKey);

  return isStarted;
}
