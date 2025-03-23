export class ApiError extends Error {
  status: number;
  json: any; // Can hold any object

  constructor(message: string, status: number, json?: any) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.json = json;
  }
}

enum Methods {
  GET = "GET",
  POST = "POST",
  PUT = "PUT",
  DELETE = "DELETE",
  PATCH = "PATCH",
  HEAD = "HEAD",
  OPTIONS = "OPTIONS"
}

const BASE_URL = "https://localhost:8080";

export async function httpGet(url: string) {
  return api(`${BASE_URL}${url}`, Methods.GET);
}

export async function httpPost<T>(url: string, body: T) {
  return api(`${BASE_URL}${url}`, Methods.POST, body);
}

async function api<T = any>(
  url: string,
  method: keyof typeof Methods,
  body?: any,
  headers?: Record<string, string>
): Promise<T> {
  const fetchOptions: RequestInit = {
    credentials: "include",
    mode: "cors",
    method: Methods[method], // Access the enum value using index access
    headers: headers
  };

  if (body) {
    fetchOptions.body = JSON.stringify(body);
    if (!headers || !headers["Content-Type"]) {
      fetchOptions.headers = {
        ...headers,
        "Content-Type": "application/json"
      };
    }
  }

  const response = await fetch(url, fetchOptions);

  if (!response.ok) {
    // Handle non-2xx responses (e.g., throw an error)
    const errorData = await response
      .json()
      .catch(() => ({ message: response.statusText }));
    throw new ApiError(
      `HTTP error ${response.status}`,
      response.status,
      errorData
    );
  }

  try {
    const data: T = await response.json();
    return data;
  } catch (error) {
    // Handle cases where the response is not JSON
    if (response.status === 204) {
      // No Content
      return null as T;
    }
    const textData = await response.text();
    console.warn("Response was not json. Returning text data.");
    return textData as T;
  }
}
