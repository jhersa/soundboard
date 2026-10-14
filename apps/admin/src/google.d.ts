// Tipos mínimos de Google Identity Services (https://accounts.google.com/gsi/client).
interface GoogleCredentialResponse {
  credential: string;
}

interface Window {
  google?: {
    accounts: {
      id: {
        initialize(options: { client_id: string; callback: (response: GoogleCredentialResponse) => void }): void;
        renderButton(element: HTMLElement, options: Record<string, unknown>): void;
      };
    };
  };
}
