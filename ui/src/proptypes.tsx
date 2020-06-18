import * as Client from './leifDbClientAPI';

export type Page = 'Home' | 'Database' | 'Admin';

export interface Server {
  address: string;
  healthy: boolean;
  client?: Client.LeifDbClientAPIContext;
}
