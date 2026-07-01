export interface GenericOAuthProvider {
  name: string;
  displayName: string;
  clientId: string;
  authUrl: string;
  scopes: string;
}
