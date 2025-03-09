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
  return api(`${BASE_URL}${url}`, Methods.POST, JSON.stringify(body));
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
    const errorData = await response.json(); // attempt to get json error response.
    throw new Error(
      `HTTP error! status: ${response.status}, message: ${JSON.stringify(
        errorData
      )}`
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
