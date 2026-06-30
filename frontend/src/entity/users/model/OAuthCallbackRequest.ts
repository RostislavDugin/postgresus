export interface OAuthCallbackRequest {
  provider: string;
  code: string;
  redirectUri: string;
}
