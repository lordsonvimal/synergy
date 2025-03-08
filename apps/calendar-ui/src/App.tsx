import type { Component } from "solid-js";
import { Router, Route } from "@solidjs/router";
import { Home } from "./components/home";
import { AuthCallback } from "./components/authCallback";

const App: Component = () => {
  return (
    <Router>
      <Route path="/" component={Home} />
      <Route path="/callback" component={AuthCallback} />
    </Router>
  );
};

export { App };
