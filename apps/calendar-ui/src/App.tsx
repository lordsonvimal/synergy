import type { Component } from "solid-js";
import { Router, Route } from "@solidjs/router";
import { Home } from "./components/home/home";
import { AuthCallback } from "./components/auth/authCallback";
import { Dashboard } from "./components/dashboard/dashboard";
import { clientRoutes } from "./services/clientRoutes";

const App: Component = () => {
  return (
    <Router>
      <Route path={clientRoutes.home} component={Home} />
      <Route path={clientRoutes.callback} component={AuthCallback} />
      <Route path={clientRoutes.dashboard} component={Dashboard} />
    </Router>
  );
};

export { App };
