const SECRET = process.env.WALKIE_SECRET || "";

export function validateToken(token: string | undefined): boolean {
  if (!SECRET) {
    return true;
  }
  return token === SECRET;
}
