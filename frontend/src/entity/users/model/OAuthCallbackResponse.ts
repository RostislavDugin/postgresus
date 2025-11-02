export interface OAuthCallbackResponse {
  userId: string;
  email: string;
  token: string;
  isNewUser: boolean;
}
