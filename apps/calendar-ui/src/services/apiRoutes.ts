export const urls = {
  authRedirect: "/api/v1/auth/redirect",
  authLogin: "/api/v1/auth/login",
  authLogout: "/api/v1/auth/logout",
  authCallback: "/api/v1/auth/callback",
  userShow: (id: string) => `/api/v1/users/${id}`
};
