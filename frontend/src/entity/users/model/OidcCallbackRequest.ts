export interface OidcCallbackRequest {
  code: string;
  state: string;
  redirectUri: string;
}
