import type { EntryKey } from './storage';

export type Routes = {
  Home: undefined;
  Draft: { id: string };
  Publish: { id: string };
  Article: EntryKey;
  Subscribe: undefined;
  Subscriptions: undefined;
  Desktop: undefined;
};

export type Tabs = { Writing: undefined; Reading: undefined; Settings: undefined };
