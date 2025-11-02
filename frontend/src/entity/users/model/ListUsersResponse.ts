import type { UserProfile } from './UserProfile';

export interface ListUsersResponse {
  users: UserProfile[];
  total: number;
}
