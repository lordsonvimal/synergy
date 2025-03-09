import type { Component } from "solid-js";
import { Router, Route } from "@solidjs/router";
import { Home } from "./components/home";
import { AuthCallback } from "./components/authCallback";
import { Dashboard } from "./components/dashboard";

const App: Component = () => {
  return (
    <Router>
      <Route path="/" component={Home} />
      <Route path="/callback" component={AuthCallback} />
      <Route path="/dashboard" component={Dashboard} />
    </Router>
  );
};

export { App };
