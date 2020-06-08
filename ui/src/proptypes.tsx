export type Page = 'Home' | 'Database' | 'Admin';

export interface AppContentProps {
  page: Page;
}

export interface Server {
  address: string;
  healthy: boolean;
}

export interface AdminPageProps {
  currentHost: Server;
  setHost: React.Dispatch<React.SetStateAction<Server>>;
}

export type Mode = 'ModeSearch' | 'ModeSet' | 'ModeDelete';

export interface DbPageProps {
  host: Server;
}

export interface HeaderProps {
  page: Page;
  clickHandler: (p: Page) => void;
}

