import * as SecureStore from 'expo-secure-store';
import { type Connection, normalizeConnection } from './desktop.ts';

const connectionKey = 'kinpage.desktop.connection.v1';
const keychainOptions = { keychainAccessible: SecureStore.WHEN_UNLOCKED_THIS_DEVICE_ONLY };

export async function getConnection(): Promise<Connection | null> {
  const raw = await SecureStore.getItemAsync(connectionKey, keychainOptions);
  if (!raw) return null;
  const { origin, username, password } = JSON.parse(raw) as Connection;
  return normalizeConnection(origin, username, password);
}

export function storeConnection(connection: Connection) {
  return SecureStore.setItemAsync(connectionKey, JSON.stringify(connection), keychainOptions);
}

export function forgetConnection() {
  return SecureStore.deleteItemAsync(connectionKey, keychainOptions);
}
