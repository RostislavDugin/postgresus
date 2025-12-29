const AUTHORIED_USER_TOKEN_KEY = 'databasus_user_token';
const AUTHORIED_USER_ID_KEY = 'databasus_user_id';

export const accessTokenHelper = {
  saveAccessToken: (token: string) => {
    if (typeof localStorage === 'undefined') {
      return;
    }

    localStorage.setItem(AUTHORIED_USER_TOKEN_KEY, token);
  },

  getAccessToken: (): string | undefined => {
    if (typeof localStorage === 'undefined') {
      return;
    }

    return localStorage.getItem(AUTHORIED_USER_TOKEN_KEY) || undefined;
  },

  cleanAccessToken: () => {
    if (typeof localStorage === 'undefined') {
      return;
    }

    localStorage.removeItem(AUTHORIED_USER_TOKEN_KEY);
  },

  saveUserId: (id: string) => {
    if (typeof localStorage === 'undefined') {
      return;
    }

    localStorage.setItem(AUTHORIED_USER_ID_KEY, id);
  },

  getUserId: (): string | undefined => {
    if (typeof localStorage === 'undefined') {
      return;
    }

    return localStorage.getItem(AUTHORIED_USER_ID_KEY) || undefined;
  },
};
