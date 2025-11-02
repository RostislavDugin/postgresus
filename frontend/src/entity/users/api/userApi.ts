import { getApplicationServer } from '../../../constants';
import RequestOptions from '../../../shared/api/RequestOptions';
import { accessTokenHelper } from '../../../shared/api/accessTokenHelper';
import { apiHelper } from '../../../shared/api/apiHelper';
import type { ChangePasswordRequest } from '../model/ChangePasswordRequest';
import type { InviteUserRequest } from '../model/InviteUserRequest';
import type { InviteUserResponse } from '../model/InviteUserResponse';
import type { IsAdminHasPasswordResponse } from '../model/IsAdminHasPasswordResponse';
import type { OAuthCallbackRequest } from '../model/OAuthCallbackRequest';
import type { OAuthCallbackResponse } from '../model/OAuthCallbackResponse';
import type { SetAdminPasswordRequest } from '../model/SetAdminPasswordRequest';
import type { SignInRequest } from '../model/SignInRequest';
import type { SignInResponse } from '../model/SignInResponse';
import type { SignUpRequest } from '../model/SignUpRequest';
import type { UpdateUserInfoRequest } from '../model/UpdateUserInfoRequest';
import type { UserProfile } from '../model/UserProfile';

const listeners: (() => void)[] = [];

const saveAuthorizedData = (accessToken: string, userId: string) => {
  accessTokenHelper.saveAccessToken(accessToken);
  accessTokenHelper.saveUserId(userId);
};

const notifyAuthListeners = () => {
  for (const listener of listeners) {
    listener();
  }
};

export const userApi = {
  async signUp(signUpRequest: SignUpRequest) {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(signUpRequest));
    return apiHelper.fetchPostRaw(`${getApplicationServer()}/api/v1/users/signup`, requestOptions);
  },

  async signIn(signInRequest: SignInRequest): Promise<SignInResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(signInRequest));

    return apiHelper
      .fetchPostJson(`${getApplicationServer()}/api/v1/users/signin`, requestOptions)
      .then((response: unknown): SignInResponse => {
        const typedResponse = response as SignInResponse;
        saveAuthorizedData(typedResponse.token, typedResponse.userId);
        notifyAuthListeners();
        return typedResponse;
      });
  },

  async isAnyUserExists(): Promise<boolean> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper
      .fetchGetJson(
        `${getApplicationServer()}/api/v1/users/is-any-user-exist`,
        requestOptions,
        true,
      )
      .then((response: unknown) => {
        const typedResponse = response as { isExist: boolean };
        return typedResponse.isExist;
      });
  },

  async isAdminHasPassword(): Promise<IsAdminHasPasswordResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(
      `${getApplicationServer()}/api/v1/users/admin/has-password`,
      requestOptions,
    );
  },

  async setAdminPassword(request: SetAdminPasswordRequest): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(
      `${getApplicationServer()}/api/v1/users/admin/set-password`,
      requestOptions,
    );
  },

  async changePassword(request: ChangePasswordRequest): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPutJson(
      `${getApplicationServer()}/api/v1/users/change-password`,
      requestOptions,
    );
  },

  async inviteUser(request: InviteUserRequest): Promise<InviteUserResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPostJson(`${getApplicationServer()}/api/v1/users/invite`, requestOptions);
  },

  async getCurrentUser(): Promise<UserProfile> {
    const requestOptions: RequestOptions = new RequestOptions();
    return apiHelper.fetchGetJson(`${getApplicationServer()}/api/v1/users/me`, requestOptions);
  },

  async updateUserInfo(request: UpdateUserInfoRequest): Promise<{ message: string }> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));
    return apiHelper.fetchPutJson(`${getApplicationServer()}/api/v1/users/me`, requestOptions);
  },

  async handleGitHubOAuth(request: OAuthCallbackRequest): Promise<OAuthCallbackResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));

    return apiHelper
      .fetchPostJson(`${getApplicationServer()}/api/v1/auth/github/callback`, requestOptions)
      .then((response: unknown): OAuthCallbackResponse => {
        const typedResponse = response as OAuthCallbackResponse;
        saveAuthorizedData(typedResponse.token, typedResponse.userId);
        notifyAuthListeners();
        return typedResponse;
      });
  },

  async handleGoogleOAuth(request: OAuthCallbackRequest): Promise<OAuthCallbackResponse> {
    const requestOptions: RequestOptions = new RequestOptions();
    requestOptions.setBody(JSON.stringify(request));

    return apiHelper
      .fetchPostJson(`${getApplicationServer()}/api/v1/auth/google/callback`, requestOptions)
      .then((response: unknown): OAuthCallbackResponse => {
        const typedResponse = response as OAuthCallbackResponse;
        saveAuthorizedData(typedResponse.token, typedResponse.userId);
        notifyAuthListeners();
        return typedResponse;
      });
  },

  isAuthorized: (): boolean => !!accessTokenHelper.getAccessToken(),

  logout: () => {
    accessTokenHelper.cleanAccessToken();
  },

  // listeners

  addAuthListener: (listener: () => void) => {
    listeners.push(listener);
  },

  removeAuthListener: (listener: () => void) => {
    listeners.splice(listeners.indexOf(listener), 1);
  },

  notifyAuthListeners: (): void => {
    for (const listener of listeners) {
      listener();
    }
  },

  saveAuthorizedData,
};
