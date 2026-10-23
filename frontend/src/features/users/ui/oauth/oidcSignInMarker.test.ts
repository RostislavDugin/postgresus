import { beforeEach, describe, expect, it } from 'vitest';

import { markOidcSignInStarted, readAndClearOidcSignInMarker } from './oidcSignInMarker';

describe('oidcSignInMarker', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('reports no marker when no sign-in was started', () => {
    expect(readAndClearOidcSignInMarker()).toBe(false);
  });

  it('reports the marker once the sign-in was started', () => {
    markOidcSignInStarted();

    expect(readAndClearOidcSignInMarker()).toBe(true);
  });

  it('clears the marker when it is read, so a later callback is not claimed', () => {
    markOidcSignInStarted();
    readAndClearOidcSignInMarker();

    expect(readAndClearOidcSignInMarker()).toBe(false);
  });

  it('ignores a marker another tab left with an unexpected value', () => {
    sessionStorage.setItem('oidcSignInStarted', 'maybe');

    expect(readAndClearOidcSignInMarker()).toBe(false);
  });
});
