// Claves de almacenamiento del navegador (spec frontend §8).
export const TOKEN_KEY = 'emp.token';
export const USER_KEY = 'emp.user';
export const ANALYSIS_KEY = 'emp.analysisId';

export interface StoredUser {
  email: string;
  name: string;
}

function safe<T>(fn: () => T, fallback: T): T {
  try {
    return fn();
  } catch {
    return fallback;
  }
}

export const storage = {
  getToken: (): string | null => safe(() => localStorage.getItem(TOKEN_KEY), null),
  setToken: (token: string): void => safe(() => localStorage.setItem(TOKEN_KEY, token), undefined),
  getUser: (): StoredUser | null =>
    safe(() => {
      const raw = localStorage.getItem(USER_KEY);
      return raw ? (JSON.parse(raw) as StoredUser) : null;
    }, null),
  setUser: (user: StoredUser): void => safe(() => localStorage.setItem(USER_KEY, JSON.stringify(user)), undefined),
  clearSession: (): void =>
    safe(() => {
      localStorage.removeItem(TOKEN_KEY);
      localStorage.removeItem(USER_KEY);
    }, undefined),
  getAnalysisId: (): string | null => safe(() => sessionStorage.getItem(ANALYSIS_KEY), null),
  setAnalysisId: (id: string): void => safe(() => sessionStorage.setItem(ANALYSIS_KEY, id), undefined),
  clearAnalysisId: (): void => safe(() => sessionStorage.removeItem(ANALYSIS_KEY), undefined),
};
